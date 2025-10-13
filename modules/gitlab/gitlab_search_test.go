package gitlab_utils

import (
	"fmt"
	"i4c/modules/gitlab/common_filters"
	"i4c/modules/gitlab/issue_filters"
	"testing"
	"time"
)

func TestGetIssuesFromGitlab(t *testing.T) {
	
	g := GitlabUtils{
		ProjectId: "2136",
		Token:     "6xuy7K7NTYSmStxkgHZM",
	}

	timeStart, _ := time.Parse(time.RFC3339, "2023-02-22T06:00:00.000+00:00")
	timeEnd, _:= time.Parse(time.RFC3339, "2023-02-22T10:00:00.000+00:00")

	dateSpec := issue_filters.DateIssueFilter{
		UpdatedAfter:  timeStart,
		UpdatedBefore: timeEnd,
	}

	statusSpec := issue_filters.StatusIssueFilter{
		Status: issue_filters.Opened,
	}

	specs := []IssueFilter{&dateSpec, &statusSpec}

	issues := g.GetIssuesFromGitlab(specs)

	if len(issues) != 1 {
		t.Fatalf("get %d opened issues instead of 1", len(issues))
	}

	statusSpec.Status = issue_filters.Closed

	specs = []IssueFilter{&dateSpec, &statusSpec}

	issues = g.GetIssuesFromGitlab(specs)

	if len(issues) != 1 {
		t.Fatalf("get %d closed issues instead of 1", len(issues))
	}

}

func TestGetTagsFromGitlab(t *testing.T) {

	orderby := common_filters.OrderByFilter{
		"updated",
	}

	pagination := common_filters.PaginationFilter{
		Page:    0,
		PerPage: 200,
	}

	specs := []TagFilter{&orderby, &pagination}

	g := GitlabUtils{
		ProjectId: "2136",
		Token:     "6xuy7K7NTYSmStxkgHZM",
	}

	tags := g.GetTagsFromGitlab(specs)

	if len(tags) < 5 {
		t.Fatalf("get %d tags, unexpected value", len(tags))
		return
	}

	fmt.Printf("get %d tags, looks ok\n", len(tags))

	fmt.Println("last tag is : " + tags[0].Name)
	fmt.Println("just before tag was : " + tags[1].Name)

}
