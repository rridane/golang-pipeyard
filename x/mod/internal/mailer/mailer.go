package mailer

import (
	"bytes"
	"github.com/go-mail/mail"
	"html/template"
)

type Config struct {
	Host       string
	Port       int
	Username   string
	Password   string
	From       string
	NoStartTLS bool
}

type Message struct {
	To       []string
	Cc       []string
	Bcc      []string
	Subject  string
	HTMLBody string
	TextBody string
}

type MailSender struct {
	dialer *mail.Dialer
	from   string
}

func New(config Config) *MailSender {
	d := mail.NewDialer(config.Host, config.Port, config.Username, config.Password)

	if config.NoStartTLS {
		d.StartTLSPolicy = mail.NoStartTLS
	}

	return &MailSender{
		dialer: d,
		from:   config.From,
	}
}

func (m *MailSender) Send(msg Message) error {
	message := mail.NewMessage()

	message.SetHeader("From", m.from)
	message.SetHeader("To", msg.To...)

	if len(msg.Cc) > 0 {
		message.SetHeader("Cc", msg.Cc...)
	}

	if len(msg.Bcc) > 0 {
		message.SetHeader("Bcc", msg.Bcc...)
	}

	message.SetHeader("Subject", msg.Subject)

	if msg.TextBody != "" {
		message.SetBody("text/plain", msg.TextBody)
	}

	if msg.HTMLBody != "" {
		if msg.TextBody == "" {
			message.SetBody("text/html", msg.HTMLBody)
		} else {
			message.AddAlternative("text/html", msg.HTMLBody)
		}
	}

	return m.dialer.DialAndSend(message)
}

func (m *MailSender) SendTemplate(
	to []string,
	subject string,
	tmpl *template.Template,
	data any,
) error {
	var body bytes.Buffer

	if err := tmpl.Execute(&body, data); err != nil {
		return err
	}

	return m.Send(Message{
		To:       to,
		Subject:  subject,
		HTMLBody: body.String(),
	})
}
