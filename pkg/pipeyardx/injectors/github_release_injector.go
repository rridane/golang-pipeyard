package injectors

import (
	"fmt"

	github_releases "github.com/rridane/golang-pipeyard/pkg/github_releases_sdk"
)

// GitHubReleaseInjector fournit une ReleaseFactory GitHub configurée explicitement.
type GitHubReleaseInjector struct {
	Token      string
	Owner      string
	Repo       string
	Enterprise bool
	BaseURL    string
}

// Inject construit et renvoie une ReleaseFactory GitHub configurée.
func (g *GitHubReleaseInjector) Inject(stageName string) *github_releases.ReleaseFactory {
	fmt.Printf("🔧 Injection de GitHubReleaseFactory pour le stage %s\n", stageName)

	if g.Token == "" || g.Owner == "" || g.Repo == "" {
		fmt.Printf("❌ Configuration GitHub incomplète pour %s (token, owner ou repo manquant)\n", stageName)
		return nil
	}

	config := github_releases.Config{
		Token:      g.Token,
		Owner:      g.Owner,
		Repo:       g.Repo,
		Enterprise: g.Enterprise,
		BaseURL:    g.BaseURL,
	}

	fmt.Printf("📋 Configuration GitHub: owner=%s, repo=%s, enterprise=%t\n",
		g.Owner, g.Repo, g.Enterprise)

	return github_releases.NewReleaseFactory(config)
}
