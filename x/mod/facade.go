package mod

import (
	"github.com/rridane/golang-pipeyard/x/mod/internal/asset_finder"
	"github.com/rridane/golang-pipeyard/x/mod/internal/container_builder"
	"github.com/rridane/golang-pipeyard/x/mod/internal/filesystem"
	"github.com/rridane/golang-pipeyard/x/mod/internal/github_releases_sdk"
	"github.com/rridane/golang-pipeyard/x/mod/internal/gitlab_finder"
	"github.com/rridane/golang-pipeyard/x/mod/internal/gitlab_finder/common_filters"
	"github.com/rridane/golang-pipeyard/x/mod/internal/gitlab_finder/issue_filters"
	"github.com/rridane/golang-pipeyard/x/mod/internal/gitlab_releases_sdk"
	"github.com/rridane/golang-pipeyard/x/mod/internal/mailer"
	"github.com/rridane/golang-pipeyard/x/mod/internal/shell_executor"
	"github.com/rridane/golang-pipeyard/x/mod/internal/teams_sdk"
	"go.uber.org/zap"
)

// ContainerBuilder

type ContainerBuilder = container_builder.ContainerBuilder
type BuildRequest = container_builder.BuildRequest
type RegistryCredential = container_builder.RegistryCredential
type Artifact = container_builder.Artifact
type Dockerfile = container_builder.Dockerfile

// FileSystem

type FileSystemProcessor = filesystem.FileSystemProcessor

// gitlabApi

type GitlabApi = gitlab_finder.GitlabUtils

type GitlabTag = gitlab_finder.Tag
type GitlabIssue = gitlab_finder.Issue
type GitlabLink = gitlab_finder.Link
type GitlabReleaseInfo = gitlab_finder.Release

type GitlabIssueFilter = gitlab_finder.IssueFilter
type GitlabTagFilter = gitlab_finder.TagFilter

type GitlabDateIssueFilter = issue_filters.DateIssueFilter
type GitlabStatusIssueFilter = issue_filters.StatusIssueFilter
type GitlabIssueStatus = issue_filters.Status

const (
	GitlabIssueClosed = issue_filters.Closed
	GitlabIssueOpened = issue_filters.Opened
)

type GitlabPaginationFilter = common_filters.PaginationFilter
type GitlabOrderByFilter = common_filters.OrderByFilter

// Gitlab

type GitlabReleaseSDK = gitlab_releases_sdk.GitlabReleaseSDK
type GitlabRelease = gitlab_releases_sdk.Release
type GitlabReleaseConfig = gitlab_releases_sdk.Config

// Shell

type ShellExecutor = shell_executor.ShellExecutor

// Teams

type TeamsSDK = teams_sdk.TeamsSDK
type TeamsNotification = teams_sdk.TeamsNotification
type TeamsNotificationButton = teams_sdk.TeamsNotificationButton

// Github

type GithubReleaseSDK = github_releases_sdk.ReleaseFactory
type GithubReleaseConfig = github_releases_sdk.Config

func NewGithubReleaseSDK(config github_releases_sdk.Config) *github_releases_sdk.ReleaseFactory {
	return github_releases_sdk.NewGithubReleaseSDK(config)
}

// Logger

type Logger = zap.Logger

type AssetFinder = asset_finder.AssetFinder

// Mail

type MailConfig = mailer.Config
type MailMessage = mailer.Message
type MailSender = mailer.MailSender

func NewMailSender(config mailer.Config) *MailSender {
	return mailer.New(config)
}
