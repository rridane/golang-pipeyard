package common_filters

import "github.com/xanzy/go-gitlab"

type OrderByFilter struct {
	OrderBy string
}

func (o *OrderByFilter) ToGitlabIssueOptions() gitlab.ListProjectIssuesOptions {

	return gitlab.ListProjectIssuesOptions{
		OrderBy: &o.OrderBy,
	}

}

func (o *OrderByFilter) ToListTagOptions() gitlab.ListTagsOptions {

	return gitlab.ListTagsOptions{
		OrderBy: &o.OrderBy,
	}

}
