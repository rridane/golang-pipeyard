package forge

import (
	"fmt"
	"github.com/rridane/golang-pipeyard/x/mod"
)

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
		fmt.Printf("GitLab not configured for %s: Token is required\n", stageName)
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
