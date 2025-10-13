package issue_filters

import (
	"github.com/xanzy/go-gitlab"
	"time"
)

type DateIssueFilter struct {
	UpdatedAfter  time.Time
	UpdatedBefore time.Time
}


func (d *DateIssueFilter) ToListProjectIssuesOptions() gitlab.ListProjectIssuesOptions {

	return gitlab.ListProjectIssuesOptions{
		UpdatedAfter: &d.UpdatedAfter,
		UpdatedBefore: &d.UpdatedBefore,
	}

}
