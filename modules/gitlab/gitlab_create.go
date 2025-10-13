package gitlab_utils

import (
	"fmt"
	"github.com/xanzy/go-gitlab"
	"i4c/modules/utils"
)

func (g *GitlabUtils) CreateTag(t Tag) Tag {

	client, err := gitlab.NewClient(g.Token, gitlab.WithBaseURL(GitlabUrl))

	fmt.Println("Creating tag " + t.Name + " ...")

	ro := &gitlab.CreateTagOptions{
		TagName: &t.TagName,
		Ref:     &t.Ref,
		Message: &t.Message,
	}

	_, _, err = client.Tags.CreateTag(g.ProjectId, ro)

	if err != nil {
		panic(err)
	}

	return t

}

func (g *GitlabUtils) CreateRelease(r Release, t Tag) Release {

	client, err := gitlab.NewClient(g.Token, gitlab.WithBaseURL(GitlabUrl))

	fmt.Println("Creating release " + r.ReleaseVersion + " ...")

	ro := &gitlab.CreateReleaseOptions{
		Name:        &r.ReleaseVersion,
		TagName:     &r.ReleaseVersion,
		TagMessage:  &t.Message,
		Description: &t.Description,
		Ref:         &t.Ref,
		Assets:      nil,
	}

	_, _, err = client.Releases.CreateRelease(g.ProjectId, ro)

	if err != nil {
		panic(err)
	}

	return r

}

func (g *GitlabUtils) CreateReleaseLink(l []Link, tag string) {

	client, err := gitlab.NewClient(g.Token, gitlab.WithBaseURL(GitlabUrl+":"+GitlabPort))

	utils.HandleErr(err, true)

	for _, li := range l {

		ltypeValue := gitlab.LinkTypeValue(li.LinkType)

		var l gitlab.CreateReleaseLinkOptions

		fmt.Println("Attaching link " + li.Title)

		l = gitlab.CreateReleaseLinkOptions{
			Name:     &li.Title,
			URL:      &li.Url,
			LinkType: &ltypeValue,
		}

		_, _, err := client.ReleaseLinks.CreateReleaseLink(g.ProjectId, tag, &l)

		utils.HandleErr(err, false)
	}
}


