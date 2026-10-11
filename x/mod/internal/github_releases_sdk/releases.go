// modules/github_releases_sdk/releases.go
package github_releases_sdk

import (
	"context"
	"fmt"
	"github.com/google/go-github/v50/github"
	"golang.org/x/oauth2"
)

// IRelease interface fluente
type IRelease interface {
	CreateTag() IRelease
	CreateRelease() IRelease
	UploadAssets(assets []string) IRelease
	Publish() IRelease
	WithChangelog() IRelease
}

// Release struct principale
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
	client          *github.Client
	created         bool
}

// Configuration GitHub
type Config struct {
	Token      string
	Owner      string
	Repo       string
	Enterprise bool
	BaseURL    string // Pour GitHub Enterprise
}

// ReleaseFactory pour la construction
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
		panic("GitHubRelease: tagName is required")
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
			panic(fmt.Sprintf("GitHubRelease: failed to create enterprise client: %v", err))
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
	if r.created {
		return r
	}

	fmt.Printf("creating tag %s on %s\n", r.TagName, r.TargetCommitish)

	tag := &github.Tag{
		Tag:     &r.TagName,
		Message: &r.Body,
		Object: &github.GitObject{
			SHA:  &r.TargetCommitish, // Dans ce cas, on utilise le commitish comme référence
			Type: github.String("commit"),
		},
	}

	ctx := context.Background()
	_, _, err := r.client.Git.CreateTag(ctx, r.Config.Owner, r.Config.Repo, tag)
	if err != nil {
		fmt.Printf("tag not created, it may already exist: %v\n", err)
		return r
	}

	ref := &github.Reference{
		Ref: github.String("refs/tags/" + r.TagName),
		Object: &github.GitObject{
			SHA:  &r.TargetCommitish,
			Type: github.String("commit"),
		},
	}

	_, _, err = r.client.Git.CreateRef(ctx, r.Config.Owner, r.Config.Repo, ref)
	if err != nil {
		fmt.Printf("ref not created, it may already exist: %v\n", err)
	}

	fmt.Printf("tag %s created\n", r.TagName)
	return r
}

func (r *Release) CreateRelease() IRelease {
	if r.created {
		return r
	}

	fmt.Printf("creating release %s\n", r.ReleaseName)

	release := &github.RepositoryRelease{
		TagName:         &r.TagName,
		TargetCommitish: &r.TargetCommitish,
		Name:            &r.ReleaseName,
		Body:            &r.Body,
		Draft:           &r.Draft,
		Prerelease:      &r.Prerelease,
	}

	ctx := context.Background()
	createdRelease, _, err := r.client.Repositories.CreateRelease(ctx, r.Config.Owner, r.Config.Repo, release)
	if err != nil {
		fmt.Printf("create the release: %v\n", err)
		return r
	}

	r.created = true
	fmt.Printf("release created: %s\n", createdRelease.GetHTMLURL())
	return r
}

func (r *Release) UploadAssets(assets []string) IRelease {
	if !r.created {
		fmt.Println("the release must be created before uploading assets")
		return r
	}

	fmt.Printf("uploading %d assets\n", len(assets))

	ctx := context.Background()

	_, _, err := r.client.Repositories.GetReleaseByTag(ctx, r.Config.Owner, r.Config.Repo, r.TagName)
	if err != nil {
		fmt.Printf("read the release: %v\n", err)
		return r
	}

	return r
}

// Publish flips the release from draft to published.
func (r *Release) Publish() IRelease {
	if !r.created {
		fmt.Println("the release must be created before publishing")
		return r
	}

	fmt.Println("📢 Publishing release...")

	ctx := context.Background()
	release, _, err := r.client.Repositories.GetReleaseByTag(ctx, r.Config.Owner, r.Config.Repo, r.TagName)
	if err != nil {
		fmt.Printf("read the release: %v\n", err)
		return r
	}

	draft := false
	update := &github.RepositoryRelease{
		Draft: &draft,
	}

	_, _, err = r.client.Repositories.EditRelease(ctx, r.Config.Owner, r.Config.Repo, release.GetID(), update)
	if err != nil {
		fmt.Printf("publish the release: %v\n", err)
		return r
	}

	fmt.Println("release published")
	return r
}

// WithChangelog is a stub: it does not read the commit log yet.
func (r *Release) WithChangelog() IRelease {
	fmt.Println("📝 Generating changelog...")

	ctx := context.Background()

	commits, _, err := r.client.Repositories.ListCommits(ctx, r.Config.Owner, r.Config.Repo, &github.CommitsListOptions{})
	if err == nil && len(commits) > 0 {
		changelog := "## Changes in this release:\n\n"
		// Prendre les 10 derniers commits comme exemple
		for i, commit := range commits {
			if i >= 10 {
				break
			}
			msg := commit.GetCommit().GetMessage()
			if len(msg) > 100 {
				msg = msg[:100] + "..."
			}
			changelog += fmt.Sprintf("- %s\n", msg)
		}
		r.Body = changelog + "\n\n" + r.Body
		fmt.Println("changelog generated")
	}

	return r
}
