package gitlab_finder

import (
	"time"
)

const GitlabUrl = "https://gitlab.com"
const GitlabPort = "443"

type IGitlabUtils interface {
	GetIssuesFromGitlab(s []IssueFilter) []Link
	GetTagsFromGitlab(t []TagFilter) []Tag
}

type GitlabUtils struct {
	ProjectId string
	Token     string
}

type Link struct {
	LinkType string
	Url      string
	Title    string
	IID      int
	Labels   []string
}

type Issue struct {
	WebUrl string
	Author string
}

type Release struct {
	ReleaseVersion string
	Tag            string
}

type Tag struct {
	Name         string
	Ref          string
	Message      string
	Description  string
	TagName      string
	CreationDate time.Time
}
