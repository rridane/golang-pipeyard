package common_filters

import "gitlab.com/gitlab-org/api/client-go"

type OrderByFilter struct {
	OrderBy string
}

func (o *OrderByFilter) ToListProjectIssuesOptions() gitlab.ListProjectIssuesOptions {

	return gitlab.ListProjectIssuesOptions{
		OrderBy: &o.OrderBy,
	}

}

func (o *OrderByFilter) ToListTagOptions() gitlab.ListTagsOptions {

	return gitlab.ListTagsOptions{
		OrderBy: &o.OrderBy,
	}

}

// Both filters satisfy IssueFilter and TagFilter, which is what makes them
// common. The issue-side method used to be named ToGitlabIssueOptions, so it
// matched neither interface and the filters could not be passed anywhere.
