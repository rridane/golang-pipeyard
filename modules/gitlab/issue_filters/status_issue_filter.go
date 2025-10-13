package issue_filters

import (
	"github.com/xanzy/go-gitlab"
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