package inj

import (
	"fmt"
	gitlab_releases_sdk "github.com/rridane/golang-pipeyard/pkg/gitlab_releases_sdk"
)

// GitlabReleaseInjector fournit une ReleaseFactory Gitlab configurée explicitement.
type GitlabReleaseInjector struct {
	Token        string
	ProjectId    string
	TeamsWebHook string
	ProjectName  string
	ReleasePage  string
}

// Inject construit et renvoie une ReleaseFactory Gitlab configurée.
func (g *GitlabReleaseInjector) Inject(stageName string) *gitlab_releases_sdk.ReleaseFactory {
	fmt.Printf("🔧 Injection de GitlabReleaseFactory pour le stage %s\n", stageName)

	if g.Token == "" {
		fmt.Printf("❌ Configuration Gitlab incomplète pour %s (token manquant)\n", stageName)
		return nil
	}

	config := gitlab_releases_sdk.Config{
		Token:        g.Token,
		ProjectId:    g.ProjectId,
		TeamsWebHook: g.TeamsWebHook,
		ReleasePage:  g.ReleasePage,
		ProjectName:  g.ProjectName,
	}

	release_factory := gitlab_releases_sdk.ReleaseFactory{Config: config}

	return &release_factory
}
