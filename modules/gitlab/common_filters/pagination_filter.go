package common_filters

import "github.com/xanzy/go-gitlab"

type PaginationFilter struct {
	Page int
	PerPage int
}

func (l *PaginationFilter) ToGitlabIssueOptions() gitlab.ListProjectIssuesOptions {

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