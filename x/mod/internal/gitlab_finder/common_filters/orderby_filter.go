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

// Both filters in this package satisfy IssueFilter and TagFilter, which is what
// makes them common: they apply to an issue query and to a tag query alike.
