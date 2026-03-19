package inj

import (
	"fmt"
	"github.com/rridane/golang-pipeyard/x/mod"
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
func (g *GitlabReleaseInjector) Inject(stageName string) *mod.GitlabRelease {
	fmt.Printf("🔧 Injection de GitlabReleaseFactory pour le stage %s\n", stageName)

	if g.Token == "" {
		fmt.Printf("❌ Configuration Gitlab incomplète pour %s (token manquant)\n", stageName)
		return nil
	}

	config := mod.GitlabReleaseConfig{
		Token:        g.Token,
		ProjectId:    g.ProjectId,
		TeamsWebHook: g.TeamsWebHook,
		ReleasePage:  g.ReleasePage,
		ProjectName:  g.ProjectName,
	}

	release := mod.GitlabRelease{Config: config}

	return &release
}
