package mail_sender

import (
	"bytes"
	"embed"
	"fmt"
	"github.com/go-mail/mail"
	"go.uber.org/zap"
	"golang.org/x/net/html"
	"html/template"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

type MailSender struct {
	logger            *zap.Logger
	smtpConf          smtpConf
	email             email
	insecure          bool
	emailAssets       embed.FS
	templateVariables map[string]any
}

type smtpConf struct {
	url      string
	port     int
	username string
	password string
}

type email struct {
	from    []string
	to      []string
	subject string
}

func main(logger zap.Logger) {
	sendKarateReportEmail()
}

func (s *mailSender) sendKarateReportEmail() {
	s.logger.Info("Try sending email")
	d := mail.NewDialer(s.smtpConf.url, s.smtpConf.port, s.smtpConf.username, s.smtpConf.password)

	if s.insecure {
		d.StartTLSPolicy = mail.NoStartTLS
	}

	m := mail.NewMessage()

	m.SetHeader("From", s.email.from...)
	m.SetHeader("To", s.email.to...)

	currentDate := time.Now().Format("02-01-2006")
	title := fmt.Sprintf("Résultat du test du %s", currentDate)
	m.SetHeader("Subject", s.email.subject)

	tmpl := template.Must(template.New("email_template.html").ParseFS(s.emailAssets, "templates/email_template.html"))

	// Prepare the email body with the data for the template
	var body bytes.Buffer

	data := struct {
		PassCount   int
		FailCount   int
		Date        string
		TotalCount  int
		ReportTable template.HTML
	}{
		PassCount:   passCount,
		FailCount:   failCount,
		TotalCount:  totalCount,
		Date:        currentDate,
		ReportTable: template.HTML(reportTable),
	}

	// Execute the template with data
	if err := tmpl.Execute(&body, data); err != nil {
		log.Fatal(err)
	}

	// Save the generated HTML to a local file
	outputPath := "result.html"
	if err := os.WriteFile(outputPath, body.Bytes(), 0644); err != nil {
		log.Fatalf("Error writing result.html: %v", err)
	}
	fmt.Printf("Template saved to %s\n", outputPath)

	// Générer une image avec Puppeteer
	imagePath := "chart.png"

	err = captureChartImage(outputPath, imagePath)
	if err != nil {
		log.Fatalf("Erreur lors de la génération de l'image : %v", err)
	}

	// Mettre à jour le fichier HTML avec l'image générée
	err = updateHTMLWithChartImage(outputPath, imagePath)

	if err != nil {
		log.Fatalf("Erreur lors de la mise à jour du fichier HTML avec l'image : %v", err)
	}

	// Reload the updated HTML file
	updatedHTML, err := os.ReadFile(outputPath)
	if err != nil {
		log.Fatalf("Error reading updated result.html: %v", err)
	}

	// Set the updated HTML as the email body
	m.SetBody("text/html", string(updatedHTML))

	// Intégrer l'image dans l'email (en pièce jointe ou inline)
	m.Embed(imagePath)

	// Send email
	if err := d.DialAndSend(m); err != nil {
		fmt.Println("Failed sending mail")
		log.Fatal(err)
	}
	fmt.Println("Mail sent without error")
}

// extractCounts extracts the pass and fail counts from the HTML content
func extractCounts(htmlContent string) (int, int) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		log.Fatal("Error parsing HTML content:", err)
	}

	var passCount, failCount int
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "id" && attr.Val == "nav-pass" {
					passCount = extractIntValue(n)
				} else if attr.Key == "id" && attr.Val == "nav-fail" {
					failCount = extractIntValue(n)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return passCount, failCount
}

// extractIntValue extracts integer value from an HTML node
func extractIntValue(n *html.Node) int {
	if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
		var value int
		fmt.Sscanf(n.FirstChild.Data, "%d", &value)
		return value
	}
	return 0
}

// extractTable extracts the first HTML table from the content
func extractTable(htmlContent string) string {

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		log.Fatal("Error parsing HTML content:", err)
	}
	var tableContent bytes.Buffer
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "table" {
			html.Render(&tableContent, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)

	return tableContent.String()

}

// addPrefixToHref adds a base URL prefix to all href attributes in the HTML content
func addPrefixToHref(htmlContent string, baseURL string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		log.Fatalf("Error parsing HTML content for href modification: %v", err)
	}

	var modifyHref func(*html.Node)
	modifyHref = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for i, attr := range n.Attr {
				if attr.Key == "href" && !strings.HasPrefix(attr.Val, "http") {
					// Prepend the base URL to the href attribute
					n.Attr[i].Val = baseURL + attr.Val
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			modifyHref(c)
		}
	}

	modifyHref(doc)

	var modifiedContent bytes.Buffer
	if err := html.Render(&modifiedContent, doc); err != nil {
		log.Fatalf("Error rendering modified HTML content: %v", err)
	}
	return modifiedContent.String()
}

func captureChartImage(htmlPath, imagePath string) error {
	fmt.Println("Génération de l'image avec Puppeteer...")

	// Écrire le fichier temporaire pour le script JS
	scriptPath, err := writeTempFile(chartScript, "generate_chart_image.js")
	if err != nil {
		return err
	}
	defer os.Remove(scriptPath) // Supprimer le fichier temporaire après utilisation

	// Exécuter le script avec Puppeteer
	cmd := exec.Command("node", scriptPath, htmlPath, imagePath)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		log.Printf("Erreur Puppeteer - sortie standard : %s\n", stderr.String())
		return fmt.Errorf("Erreur Puppeteer : %v", err)
	}

	fmt.Printf("Image générée : %s\n", imagePath)
	return nil
}

func updateHTMLWithChartImage(htmlPath, imagePath string) error {
	// Lire le contenu du fichier HTML existant
	htmlContent, err := os.ReadFile(htmlPath)
	if err != nil {
		return fmt.Errorf("Erreur lors de la lecture du fichier HTML : %v", err)
	}

	// Modifier le contenu pour remplacer le <canvas> par une <img>
	updatedContent := strings.ReplaceAll(
		string(htmlContent),
		`<canvas id="resultChart"></canvas>`,
		fmt.Sprintf(`<img src="%s" alt="Résultats des tests" />`, imagePath),
	)

	fmt.Println(updatedContent)

	// Réécrire le fichier HTML avec les modifications
	err = os.WriteFile(htmlPath, []byte(updatedContent), 0644)
	if err != nil {
		return fmt.Errorf("Erreur lors de l'écriture du fichier HTML mis à jour : %v", err)
	}

	fmt.Printf("Fichier HTML mis à jour avec l'image : %s\n", imagePath)
	return nil
}

func writeTempFile(content []byte, filename string) (string, error) {
	tempDir := os.TempDir()
	tempPath := fmt.Sprintf("%s/%s", tempDir, filename)
	err := os.WriteFile(tempPath, content, 0644)
	if err != nil {
		return "", fmt.Errorf("Erreur lors de la création du fichier temporaire : %v", err)
	}
	return tempPath, nil
}
