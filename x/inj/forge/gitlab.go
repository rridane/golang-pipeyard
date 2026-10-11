package forge

import (
	"fmt"
	"github.com/rridane/golang-pipeyard/x/mod"
)

// GitlabReleaseInjector provides an explicitly configured GitLab ReleaseFactory.
type GitlabReleaseInjector struct {
	Token        string
	ProjectId    string
	TeamsWebHook string
	ProjectName  string
	ReleasePage  string
}

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
