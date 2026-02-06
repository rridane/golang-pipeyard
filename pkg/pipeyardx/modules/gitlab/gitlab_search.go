package gitlab_utils

/*
	Get Infos from gitlab => Issues, tags
	Send infos to gitlab => Commits
*/

import (
	"encoding/json"
	"fmt"
	"gitlab.com/gitlab-org/api/client-go"
)

type IssueFilter interface {
	ToListProjectIssuesOptions() gitlab.ListProjectIssuesOptions
}

func (g *GitlabUtils) GetIssuesFromGitlab(s []IssueFilter) []Link {

	client, _ := gitlab.NewClient(g.Token, gitlab.WithBaseURL(GitlabUrl+":"+GitlabPort))

	var listIssueOptions gitlab.ListProjectIssuesOptions

	for _, is := range s {
		gi := is.ToListProjectIssuesOptions()

		gim, _ := json.Marshal(gi)

		_ = json.Unmarshal(gim, &listIssueOptions)

	}

	issues, _, _ := client.Issues.ListProjectIssues(g.ProjectId, &listIssueOptions)

	var links []Link

	for _, issue := range issues {

		links = append(links, Link{
			LinkType: "other",
			Url:      issue.WebURL,
			Title:    issue.Title,
			IID:      issue.IID,
			Labels:   issue.Labels,
		})

	}

	return links

}

type TagFilter interface {
	ToListTagOptions() gitlab.ListTagsOptions
}

func (g *GitlabUtils) GetTagsFromGitlab(t []TagFilter) []Tag {

	client, err := gitlab.NewClient(g.Token, gitlab.WithBaseURL(GitlabUrl+":"+GitlabPort))

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Getting Tags From Gitlab ...")

	var listTagOptions gitlab.ListTagsOptions

	for _, ts := range t {
		to := ts.ToListTagOptions()

		tom, _ := json.Marshal(to)

		_ = json.Unmarshal(tom, &listTagOptions)

	}

	tagsFromGitlab, _, err := client.Tags.ListTags(g.ProjectId, &listTagOptions)

	if err != nil {
		fmt.Println(err)
	}

	var tags []Tag

	for _, tag := range tagsFromGitlab {

		t := Tag{
			Name:         tag.Name,
			Ref:          tag.Target,
			Message:      tag.Message,
			Description:  tag.Message,
			TagName:      tag.Name,
			CreationDate: *tag.Commit.CommittedDate,
		}

		tags = append(tags, t)

	}

	return tags
}

func (g *GitlabUtils) GetReleaseFromGitlab(releaseVersion string) Release {

	fmt.Println("Searching Release " + releaseVersion)

	client, err := gitlab.NewClient(g.Token, gitlab.WithBaseURL(GitlabUrl+":"+GitlabPort))

	if err != nil {
		fmt.Println(err)
	}

	rel, _, err := client.Releases.GetRelease(g.ProjectId, releaseVersion)

	if rel == nil {
		return Release{
			ReleaseVersion: "",
			Tag:            "",
		}
	}

	return Release{
		ReleaseVersion: rel.TagName,
		Tag:            rel.TagName,
	}

}
