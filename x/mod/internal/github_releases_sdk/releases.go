// modules/github_releases_sdk/releases.go
package github_releases_sdk

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/go-github/v50/github"
	"golang.org/x/oauth2"
)

// IRelease is a fluent chain. Every step short-circuits once one has failed,
// and Err reports the first failure — the chain cannot return an error itself.
type IRelease interface {
	CreateTag() IRelease
	CreateRelease() IRelease
	UploadAssets(assets []string) IRelease
	Publish() IRelease
	WithChangelog() IRelease
	Err() error
}

// Release is a release in the middle of being made.
type Release struct {
	TagName     string
	ReleaseName string
	// TargetCommitish specifies the branch name or commit SHA the tag should point to.
	TargetCommitish string
	Body            string
	Draft           bool
	Prerelease      bool
	Assets          []string
	Config          Config

	// URL is the published release page, available once CreateRelease ran.
	URL string

	client  *github.Client
	created bool
	err     error
}

func (r *Release) Err() error { return r.err }

// fail records the first failure. Later steps see it and do nothing.
func (r *Release) fail(format string, args ...any) IRelease {
	if r.err == nil {
		r.err = fmt.Errorf(format, args...)
	}
	return r
}

// alreadyExists tells a re-run apart from a real failure: GitHub answers 422
// when the tag or the ref is already there, which is fine when a release is
// replayed. Anything else is a failure.
func (r *Release) alreadyExists(err error) bool {
	var response *github.ErrorResponse
	if !errors.As(err, &response) || response.Response == nil {
		return false
	}
	return response.Response.StatusCode == http.StatusUnprocessableEntity
}

// Configuration GitHub
type Config struct {
	Token      string
	Owner      string
	Repo       string
	Enterprise bool
	BaseURL    string // for GitHub Enterprise
}

// ReleaseFactory collects the release settings before it is prepared.
type ReleaseFactory struct {
	tagName         string
	releaseName     string
	targetCommitish string
	body            string
	draft           bool
	prerelease      bool
	config          Config
}

func NewGithubReleaseSDK(config Config) *ReleaseFactory {
	return &ReleaseFactory{
		targetCommitish: "main",
		draft:           true,
		config:          config,
	}
}

func (f *ReleaseFactory) WithTag(tagName string) *ReleaseFactory {
	f.tagName = tagName
	return f
}

func (f *ReleaseFactory) WithReleaseName(name string) *ReleaseFactory {
	f.releaseName = name
	return f
}

func (f *ReleaseFactory) WithTargetBranch(branch string) *ReleaseFactory {
	f.targetCommitish = branch
	return f
}

func (f *ReleaseFactory) WithBody(body string) *ReleaseFactory {
	f.body = body
	return f
}

func (f *ReleaseFactory) AsDraft(draft bool) *ReleaseFactory {
	f.draft = draft
	return f
}

func (f *ReleaseFactory) AsPrerelease(pre bool) *ReleaseFactory {
	f.prerelease = pre
	return f
}

// PrepareRelease validates the factory before any call is made.
func (f *ReleaseFactory) PrepareRelease() IRelease {
	if f.tagName == "" {
		return (&Release{}).fail("github release: a tag name is required")
	}
	if f.releaseName == "" {
		f.releaseName = f.tagName
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: f.config.Token},
	)
	tc := oauth2.NewClient(ctx, ts)

	var client *github.Client
	if f.config.Enterprise && f.config.BaseURL != "" {
		var err error
		client, err = github.NewEnterpriseClient(f.config.BaseURL, "", tc)
		if err != nil {
			return (&Release{}).fail("github release: enterprise client for %s: %w", f.config.BaseURL, err)
		}
	} else {
		client = github.NewClient(tc)
	}

	return &Release{
		TagName:         f.tagName,
		ReleaseName:     f.releaseName,
		TargetCommitish: f.targetCommitish,
		Body:            f.body,
		Draft:           f.draft,
		Prerelease:      f.prerelease,
		Config:          f.config,
		client:          client,
	}
}

func (r *Release) CreateTag() IRelease {
	if r.err != nil || r.created {
		return r
	}

	ctx := context.Background()

	tag := &github.Tag{
		Tag:     &r.TagName,
		Message: &r.Body,
		Object: &github.GitObject{
			SHA:  &r.TargetCommitish,
			Type: github.String("commit"),
		},
	}

	if _, _, err := r.client.Git.CreateTag(ctx, r.Config.Owner, r.Config.Repo, tag); err != nil {
		if !r.alreadyExists(err) {
			return r.fail("create the tag %s: %w", r.TagName, err)
		}
	}

	ref := &github.Reference{
		Ref: github.String("refs/tags/" + r.TagName),
		Object: &github.GitObject{
			SHA:  &r.TargetCommitish,
			Type: github.String("commit"),
		},
	}

	if _, _, err := r.client.Git.CreateRef(ctx, r.Config.Owner, r.Config.Repo, ref); err != nil {
		if !r.alreadyExists(err) {
			return r.fail("create the ref for tag %s: %w", r.TagName, err)
		}
	}

	return r
}

func (r *Release) CreateRelease() IRelease {
	if r.err != nil || r.created {
		return r
	}

	release := &github.RepositoryRelease{
		TagName:         &r.TagName,
		TargetCommitish: &r.TargetCommitish,
		Name:            &r.ReleaseName,
		Body:            &r.Body,
		Draft:           &r.Draft,
		Prerelease:      &r.Prerelease,
	}

	created, _, err := r.client.Repositories.CreateRelease(
		context.Background(), r.Config.Owner, r.Config.Repo, release)
	if err != nil {
		return r.fail("create the release %s: %w", r.ReleaseName, err)
	}

	r.created = true
	r.URL = created.GetHTMLURL()

	return r
}

// UploadAssets attaches files to the release.
func (r *Release) UploadAssets(assets []string) IRelease {
	if r.err != nil {
		return r
	}
	if !r.created {
		return r.fail("upload assets: the release must be created first")
	}

	ctx := context.Background()

	release, _, err := r.client.Repositories.GetReleaseByTag(
		ctx, r.Config.Owner, r.Config.Repo, r.TagName)
	if err != nil {
		return r.fail("read the release %s: %w", r.TagName, err)
	}

	for _, asset := range assets {
		if err := r.uploadAsset(ctx, release.GetID(), asset); err != nil {
			return r.fail("upload %s: %w", asset, err)
		}
	}

	return r
}

func (r *Release) uploadAsset(ctx context.Context, releaseID int64, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	options := &github.UploadOptions{Name: filepath.Base(path)}

	_, _, err = r.client.Repositories.UploadReleaseAsset(
		ctx, r.Config.Owner, r.Config.Repo, releaseID, options, file)

	return err
}

// Publish flips the release from draft to published.
func (r *Release) Publish() IRelease {
	if r.err != nil {
		return r
	}
	if !r.created {
		return r.fail("publish: the release must be created first")
	}

	ctx := context.Background()

	release, _, err := r.client.Repositories.GetReleaseByTag(
		ctx, r.Config.Owner, r.Config.Repo, r.TagName)
	if err != nil {
		return r.fail("read the release %s: %w", r.TagName, err)
	}

	draft := false

	if _, _, err := r.client.Repositories.EditRelease(
		ctx, r.Config.Owner, r.Config.Repo, release.GetID(),
		&github.RepositoryRelease{Draft: &draft}); err != nil {
		return r.fail("publish the release %s: %w", r.TagName, err)
	}

	return r
}

// WithChangelog prepends the last commits to the release body. It lists the
// repository's commits, not the range since the previous release, so the entries
// are the latest ten rather than the ones this release introduces.
func (r *Release) WithChangelog() IRelease {
	if r.err != nil {
		return r
	}

	commits, _, err := r.client.Repositories.ListCommits(
		context.Background(), r.Config.Owner, r.Config.Repo, &github.CommitsListOptions{})
	if err != nil {
		return r.fail("list the commits of %s/%s: %w", r.Config.Owner, r.Config.Repo, err)
	}

	if len(commits) == 0 {
		return r
	}

	changelog := "## Changes in this release:\n\n"

	for i, commit := range commits {
		if i >= changelogLength {
			break
		}
		changelog += fmt.Sprintf("- %s\n", summarise(commit.GetCommit().GetMessage()))
	}

	r.Body = changelog + "\n\n" + r.Body

	return r
}

const (
	changelogLength     = 10
	changelogLineLength = 100
)

func summarise(message string) string {
	if len(message) <= changelogLineLength {
		return message
	}
	return message[:changelogLineLength] + "..."
}
