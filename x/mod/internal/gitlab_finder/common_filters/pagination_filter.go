package common_filters

import "gitlab.com/gitlab-org/api/client-go"

type PaginationFilter struct {
	Page    int
	PerPage int
}

func (l *PaginationFilter) ToListProjectIssuesOptions() gitlab.ListProjectIssuesOptions {

	return gitlab.ListProjectIssuesOptions{
		ListOptions: gitlab.ListOptions{
			Page:    l.Page,
			PerPage: l.PerPage,
		},
	}

}

func (l *PaginationFilter) ToListTagOptions() gitlab.ListTagsOptions {

	return gitlab.ListTagsOptions{
		ListOptions: gitlab.ListOptions{
			Page:    l.Page,
			PerPage: l.PerPage,
		},
	}

}
