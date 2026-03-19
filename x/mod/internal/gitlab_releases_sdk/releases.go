/*
Manage Release struct
A Release struct is a conceptional representation of a Release in the sense of gitlab
So it's a tag, a gitlab Release (Release version) and links attached to it
*/

package gitlab_releases_sdk

import (
	"fmt"
	"github.com/rridane/golang-pipeyard/x/mod"
	"github.com/rridane/golang-pipeyard/x/mod/internal/gitlab_finder"
	"github.com/rridane/golang-pipeyard/x/mod/internal/gitlab_finder/issue_filters"
	"sort"
	"strconv"
	"time"
)

var defaultBuildTitle = func(i gitlab_finder.Link) string {
	return strconv.Itoa(i.IID) + "-" + i.Title
}

type User struct {
	UserId      string
	Username    string
	ProfilePath string
}

type IRelease interface {
	CreateTag() *Release
	CreateRelease() *Release
	AttachClosedIssuesSinceLastRelease() *Release
	CommunicateThroughTeams() *Release
	AttachAdditionalLinks() *Release
	WithCustomLinkTitle(buildTitle func(link gitlab_finder.Link) string) *Release
}

type Release struct {
	ReleaseVersion  *string
	Tag             string
	AdditionalLinks []gitlab_finder.Link
	Description     *string
	Message         *string
	BranchName      *string
	Config          Config
	LastReleaseDate time.Time
	Issues          []gitlab_finder.Link
	Exist           bool
	ExistChecked    bool
	BuildTitle      func(link gitlab_finder.Link) string
}

type Config struct {
	Token        string
	ProjectId    string
	TeamsWebHook string
	ReleasePage  string
	ProjectName  string
}

/*
Create tag from Release struct
*/
func (r *Release) CreateTag() *Release {

	if r.exists() {
		return r
	}

	g := gitlab_finder.GitlabUtils{ProjectId: r.Config.ProjectId, Token: r.Config.Token}

	t := gitlab_finder.Tag{
		Name:    r.Tag,
		Message: *r.Message,
		Ref:     *r.BranchName,
		TagName: r.Tag,
	}

	g.CreateTag(t)

	return r

}

// CreateRelease
/*
	Create Release from Release struct
*/
func (r *Release) CreateRelease() *Release {

	if r.exists() {
		return r
	}

	g := gitlab_finder.GitlabUtils{ProjectId: r.Config.ProjectId, Token: r.Config.Token}

	release := gitlab_finder.Release{
		ReleaseVersion: *r.ReleaseVersion,
		Tag:            r.Tag,
	}

	tag := gitlab_finder.Tag{
		Name:        r.Tag,
		Ref:         *r.BranchName,
		Message:     *r.Message,
		Description: *r.Description,
		TagName:     r.Tag,
	}

	g.CreateRelease(release, tag)

	return r

}

/*
Attach Issues since last Release from Release struct
*/
func (r *Release) AttachClosedIssuesSinceLastRelease() *Release {

	if r.exists() {
		return r
	}

	fmt.Println("Attaching closed issues since " + r.LastReleaseDate.Format(time.RFC1123) + " ...")

	dateSpec := issue_filters.DateIssueFilter{
		UpdatedAfter:  r.LastReleaseDate,
		UpdatedBefore: time.Now(),
	}

	statusSpec := issue_filters.StatusIssueFilter{Status: issue_filters.Closed}

	spec := []gitlab_finder.IssueFilter{&dateSpec, &statusSpec}

	g := gitlab_finder.GitlabUtils{ProjectId: r.Config.ProjectId, Token: r.Config.Token}

	issues := g.GetIssuesFromGitlab(spec)

	var links []gitlab_finder.Link

	if r.BuildTitle == nil {
		r.BuildTitle = defaultBuildTitle
	}

	for _, issue := range issues {

		links = append(links, gitlab_finder.Link{
			LinkType: "other",
			Url:      issue.Url,
			Title:    r.BuildTitle(issue),
		})

	}

	r.Issues = links

	r.Issues = sortLinks(r.Issues)

	if len(links) > 0 {

		g := gitlab_finder.GitlabUtils{ProjectId: r.Config.ProjectId, Token: r.Config.Token}

		g.CreateReleaseLink(links, r.Tag)

		return r
	}

	fmt.Println("No issue closed to attach ...")

	return r

}

func sortLinks(links []gitlab_finder.Link) []gitlab_finder.Link {
	sort.SliceStable(links, func(i, j int) bool {
		return links[i].Title < links[j].Title
	})

	return links
}

/*
Attach additional links from struct Release
*/
func (r *Release) AttachAdditionalLinks() *Release {

	if r.exists() {
		return r
	}

	fmt.Println("Attaching additional links ...")

	g := gitlab_finder.GitlabUtils{ProjectId: r.Config.ProjectId, Token: r.Config.Token}

	g.CreateReleaseLink(r.AdditionalLinks, r.Tag)

	return r

}

/*
 */
func (r *Release) CommunicateThroughTeams() *Release {

	if r.exists() {
		return r
	}

	t := mod.TeamsSDK{WebHook: r.Config.TeamsWebHook}

	b1 := mod.TeamsNotificationButton{
		Title: "Release Page",
		Url:   r.Config.ReleasePage,
	}

	b2 := mod.TeamsNotificationButton{
		Title: "Environments",
		Url:   "",
	}

	Text := "<br>Corrections ou évolutions embarquées :<br>"

	for _, link := range r.Issues {
		Text += "<br><a href='" + link.Url + "'>" + link.Title + "</a>"
	}

	notification := mod.TeamsNotification{
		Summary:             "Release " + *r.ReleaseVersion,
		Title:               "Nouvelle Release -- " + r.Config.ProjectName + " : " + *r.ReleaseVersion,
		Text:                Text,
		NotificationButtons: []mod.TeamsNotificationButton{b1, b2},
	}

	t.SendNotification(notification)

	return r

}

func (r *Release) WithCustomLinkTitle(buildTitle func(link gitlab_finder.Link) string) *Release {

	r.BuildTitle = buildTitle

	return r

}

func (r *Release) exists() bool {

	if r.ExistChecked {
		return r.Exist
	}

	fmt.Println("Checking if exists ...")

	g := gitlab_finder.GitlabUtils{ProjectId: r.Config.ProjectId, Token: r.Config.Token}

	var release gitlab_finder.Release

	release = g.GetReleaseFromGitlab(*r.ReleaseVersion)

	r.Exist = release.ReleaseVersion != ""

	r.ExistChecked = true

	if r.Exist {
		fmt.Println("Release Exist, skipping ...")
	}

	return r.Exist

}
