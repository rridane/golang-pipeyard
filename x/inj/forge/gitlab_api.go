package forge

import (
	"fmt"

	"github.com/rridane/golang-pipeyard/x/mod"
)

// GitlabApiInjector provides the query client a changelog is built from.
type GitlabApiInjector struct {
	ProjectId string
	Token     string
}

func (i *GitlabApiInjector) Inject(stageName string) *mod.GitlabApi {
	if i.ProjectId == "" || i.Token == "" {
		fmt.Printf("GitLab API not configured for %s: ProjectId and Token are both required\n", stageName)
		return nil
	}

	return &mod.GitlabApi{ProjectId: i.ProjectId, Token: i.Token}
}
