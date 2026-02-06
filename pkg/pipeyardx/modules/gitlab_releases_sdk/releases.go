/*
Manage release struct
A release struct is a conceptional representation of a release in the sense of gitlab
So it's a tag, a gitlab release (release version) and links attached to it
*/

package pipeyardx

import (
	"fmt"
	gitlab_utils "github.com/rridane/golang-pipeyard/pkg/gitlab"
	"github.com/rridane/golang-pipeyard/pkg/gitlab/issue_filters"
	teams "github.com/rridane/golang-pipeyard/pkg/teams_sdk"
	"sort"
	"strconv"
	"time"
)

var defaultBuildTitle = func(i gitlab_utils.Link) string {
	return strconv.Itoa(i.IID) + "-" + i.Title
}

type User struct {
	UserId      string
	Username    string
	ProfilePath string
}

type IRelease interface {
	CreateTag() *release
	CreateRelease() *release
	AttachClosedIssuesSinceLastRelease() *release
	CommunicateThroughTeams() *release
	AttachAdditionalLinks() *release
	WithCustomLinkTitle(buildTitle func(link gitlab_utils.Link) string) *release
}

type release struct {
	ReleaseVersion  *string
	Tag             string
	AdditionalLinks []gitlab_utils.Link
	Description     *string
	Message         *string
	BranchName      *string
	Config          Config
	LastReleaseDate time.Time
	Issues          []gitlab_utils.Link
	exist           bool
	existChecked    bool
	BuildTitle      func(link gitlab_utils.Link) string
}

type Config struct {
	Token        string
	ProjectId    string
	TeamsWebHook string
	ReleasePage  string
	ProjectName  string
}

/*
Create tag from release struct
*/
func (r *release) CreateTag() *release {

	if r.exists() {
		return r
	}

	g := gitlab_utils.GitlabUtils{ProjectId: r.Config.ProjectId, Token: r.Config.Token}

	t := gitlab_utils.Tag{
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
	Create release from release struct
*/
func (r *release) CreateRelease() *release {

	if r.exists() {
		return r
	}

	g := gitlab_utils.GitlabUtils{ProjectId: r.Config.ProjectId, Token: r.Config.Token}

	release := gitlab_utils.Release{
		ReleaseVersion: *r.ReleaseVersion,
		Tag:            r.Tag,
	}

	tag := gitlab_utils.Tag{
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
Attach Issues since last release from release struct
*/
func (r *release) AttachClosedIssuesSinceLastRelease() *release {

	if r.exists() {
		return r
	}

	fmt.Println("Attaching closed issues since " + r.LastReleaseDate.Format(time.RFC1123) + " ...")

	dateSpec := issue_filters.DateIssueFilter{
		UpdatedAfter:  r.LastReleaseDate,
		UpdatedBefore: time.Now(),
	}

	statusSpec := issue_filters.StatusIssueFilter{Status: issue_filters.Closed}

	spec := []gitlab_utils.IssueFilter{&dateSpec, &statusSpec}

	g := gitlab_utils.GitlabUtils{ProjectId: r.Config.ProjectId, Token: r.Config.Token}

	issues := g.GetIssuesFromGitlab(spec)

	var links []gitlab_utils.Link

	if r.BuildTitle == nil {
		r.BuildTitle = defaultBuildTitle
	}

	for _, issue := range issues {

		links = append(links, gitlab_utils.Link{
			LinkType: "other",
			Url:      issue.Url,
			Title:    r.BuildTitle(issue),
		})

	}

	r.Issues = links

	r.Issues = sortLinks(r.Issues)

	if len(links) > 0 {

		g := gitlab_utils.GitlabUtils{ProjectId: r.Config.ProjectId, Token: r.Config.Token}

		g.CreateReleaseLink(links, r.Tag)

		return r
	}

	fmt.Println("No issue closed to attach ...")

	return r

}

func sortLinks(links []gitlab_utils.Link) []gitlab_utils.Link {
	sort.SliceStable(links, func(i, j int) bool {
		return links[i].Title < links[j].Title
	})

	return links
}

/*
Attach additional links from struct release
*/
func (r *release) AttachAdditionalLinks() *release {

	if r.exists() {
		return r
	}

	fmt.Println("Attaching additional links ...")

	g := gitlab_utils.GitlabUtils{ProjectId: r.Config.ProjectId, Token: r.Config.Token}

	g.CreateReleaseLink(r.AdditionalLinks, r.Tag)

	return r

}

/*
 */
func (r *release) CommunicateThroughTeams() *release {

	if r.exists() {
		return r
	}

	t := teams.TeamsUtils{WebHook: r.Config.TeamsWebHook}

	b1 := teams.NotificationButton{
		Title: "release Page",
		Url:   r.Config.ReleasePage,
	}

	b2 := teams.NotificationButton{
		Title: "Environments",
		Url:   "",
	}

	Text := "<br>Corrections ou évolutions embarquées :<br>"

	for _, link := range r.Issues {
		Text += "<br><a href='" + link.Url + "'>" + link.Title + "</a>"
	}

	notification := teams.Notification{
		Summary:             "release " + *r.ReleaseVersion,
		Title:               "Nouvelle release -- " + r.Config.ProjectName + " : " + *r.ReleaseVersion,
		Text:                Text,
		NotificationButtons: []teams.NotificationButton{b1, b2},
	}

	t.SendNotification(notification)

	return r

}

func (r *release) WithCustomLinkTitle(buildTitle func(link gitlab_utils.Link) string) *release {

	r.BuildTitle = buildTitle

	return r

}

func (r *release) exists() bool {

	if r.existChecked {
		return r.exist
	}

	fmt.Println("Checking if exists ...")

	g := gitlab_utils.GitlabUtils{ProjectId: r.Config.ProjectId, Token: r.Config.Token}

	var release gitlab_utils.Release

	release = g.GetReleaseFromGitlab(*r.ReleaseVersion)

	r.exist = release.ReleaseVersion != ""

	r.existChecked = true

	if r.exist {
		fmt.Println("release exist, skipping ...")
	}

	return r.exist

}
