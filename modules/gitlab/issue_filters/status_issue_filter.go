package issue_filters

import (
	"gitlab.com/gitlab-org/api/client-go"
)

type Status string

const (
	Closed Status = "closed"
	Opened Status = "opened"
)

type StatusIssueFilter struct {
	Status Status
}

func (s *StatusIssueFilter) ToListProjectIssuesOptions() gitlab.ListProjectIssuesOptions {

	status := string(s.Status)

	return gitlab.ListProjectIssuesOptions{State: &status}

}
