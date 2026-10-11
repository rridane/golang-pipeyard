package forge

import (
	"fmt"
	"github.com/rridane/golang-pipeyard/x/mod"
)

type GitHubReleaseInjector struct {
	Token      string
	Owner      string
	Repo       string
	Enterprise bool
	BaseURL    string
}

func (g *GitHubReleaseInjector) Inject(stageName string) *mod.GithubReleaseSDK {
	fmt.Printf("🔧 Injection de GitHubReleaseFactory pour le stage %s\n", stageName)

	if g.Token == "" || g.Owner == "" || g.Repo == "" {
		fmt.Printf("GitHub not configured for %s: Token, Owner and Repo are all required\n", stageName)
		return nil
	}

	config := mod.GithubReleaseConfig{
		Token:      g.Token,
		Owner:      g.Owner,
		Repo:       g.Repo,
		Enterprise: g.Enterprise,
		BaseURL:    g.BaseURL,
	}

	fmt.Printf("📋 Configuration GitHub: owner=%s, repo=%s, enterprise=%t\n",
		g.Owner, g.Repo, g.Enterprise)

	return mod.NewGithubReleaseSDK(config)
}
