package github_releases_sdk

import (
	"github.com/rridane/golang-pipeyard/x/mod/internal/gitlab_finder"
	"github.com/rridane/golang-pipeyard/x/mod/internal/gitlab_releases_sdk"
	"log"
	"time"
)

type GitlabReleaseSDK struct {
	ReleaseVersion  *string
	Tag             string
	AdditionalLinks []gitlab_finder.Link
	Description     *string
	Message         *string
	BranchName      *string
	Config          gitlab_releases_sdk.Config
	LastReleaseDate time.Time
	Issues          []gitlab_finder.Link
	exist           bool
	existChecked    bool
}

func (r *GitlabReleaseSDK) PrepareRelease() gitlab_releases_sdk.IRelease {

	switch {
	case r.ReleaseVersion == nil:
		log.Fatal("ReleaseVersion should not be empty ...")
	case r.Tag == "":
		log.Fatal("Tag should not be empty ...")
	case *r.Description == "":
		log.Println("ShortDescription is empty, initialize it with releaseVersion ...")
		r.Description = r.ReleaseVersion
	case *r.Message == "":
		log.Println("Message is empty, initialize it with releaseVersion ...")
		r.Message = r.ReleaseVersion
	case r.LastReleaseDate.String() == "":
		log.Fatal("lastReleaseDate should not empty ...")
	case r.Config.Token == "":
		log.Fatal("Token cannot be empty to link gitlab project ...")
	case r.Config.TeamsWebHook == "":
		log.Println("TeamsWebHook is empty, please do not use CommunicateThroughTeams method ...")
	}

	release := gitlab_releases_sdk.Release{
		ReleaseVersion:  r.ReleaseVersion,
		Tag:             r.Tag,
		AdditionalLinks: r.AdditionalLinks,
		Description:     r.Description,
		Message:         r.Message,
		BranchName:      r.BranchName,
		Config:          r.Config,
		LastReleaseDate: r.LastReleaseDate,
		Issues:          r.Issues,
		Exist:           r.exist,
		ExistChecked:    r.existChecked,
	}

	return &release
}
