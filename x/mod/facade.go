package mod

import (
	"github.com/rridane/golang-pipeyard/x/mod/internal/asset_finder"
	"github.com/rridane/golang-pipeyard/x/mod/internal/container_builder"
	"github.com/rridane/golang-pipeyard/x/mod/internal/filesystem"
	"github.com/rridane/golang-pipeyard/x/mod/internal/github_releases_sdk"
	"github.com/rridane/golang-pipeyard/x/mod/internal/gitlab_finder"
	"github.com/rridane/golang-pipeyard/x/mod/internal/gitlab_releases_sdk"
	"github.com/rridane/golang-pipeyard/x/mod/internal/mailer"
	"github.com/rridane/golang-pipeyard/x/mod/internal/shell_executor"
	"github.com/rridane/golang-pipeyard/x/mod/internal/teams_sdk"
	"go.uber.org/zap"
)

// ContainerBuilder

type ContainerBuilder = container_builder.ContainerBuilder
type ContainerSpec = container_builder.ContainerSpec

type ContainerExportOptions = container_builder.ExportOptions

const (
	ContainerModeExportPush   container_builder.ExportMode = container_builder.ExportPush
	ContainerModeExportDocker container_builder.ExportMode = container_builder.ExportDocker
	ContainerModeExportOCI    container_builder.ExportMode = container_builder.ExportOCI
	ContainerModeExportLocal  container_builder.ExportMode = container_builder.ExportLocal
)

// FileSystem

type FileSystemProcessor = filesystem.FileSystemProcessor

// gitlabApi

type GitlabApi = gitlab_finder.GitlabUtils

// Gitlab

type GitlabReleaseSDK = gitlab_releases_sdk.GitlabReleaseSDK
type GitlabRelease = gitlab_releases_sdk.Release
type GitlabReleaseConfig = gitlab_releases_sdk.Config

// Shell

type ShellExecutor = shell_executor.ShellExecutor

// Teams

type TeamsSDK = teams_sdk.TeamsSDK

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
