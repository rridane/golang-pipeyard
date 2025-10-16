package issue_filters

import (
	"gitlab.com/gitlab-org/api/client-go"
	"time"
)

type DateIssueFilter struct {
	UpdatedAfter  time.Time
	UpdatedBefore time.Time
}

func (d *DateIssueFilter) ToListProjectIssuesOptions() gitlab.ListProjectIssuesOptions {

	return gitlab.ListProjectIssuesOptions{
		UpdatedAfter:  &d.UpdatedAfter,
		UpdatedBefore: &d.UpdatedBefore,
	}

}
