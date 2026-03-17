package pipeyardx

import (
	"github.com/rridane/golang-pipeyard/pkg/pipeyardx/modules/container_builder"
	"github.com/rridane/golang-pipeyard/pkg/pipeyardx/modules/file_helper"
	"github.com/rridane/golang-pipeyard/pkg/pipeyardx/modules/github_releases_sdk"
	"github.com/rridane/golang-pipeyard/pkg/pipeyardx/modules/gitlab_finder"
	"github.com/rridane/golang-pipeyard/pkg/pipeyardx/modules/gitlab_releases_sdk"
	"github.com/rridane/golang-pipeyard/pkg/pipeyardx/modules/mail_sender"
	"github.com/rridane/golang-pipeyard/pkg/pipeyardx/modules/shell_executor"
	"github.com/rridane/golang-pipeyard/pkg/pipeyardx/modules/teams_sdk"
	"go.uber.org/zap"
)

type ContainerBuilder = container_builder.ContainerBuilder
type FileHelper = file_helper.FileHelper

type GitlabApi = gitlab_finder.GitlabUtils

type GitlabReleaseSDK = gitlab_releases_sdk.ReleaseFactory

type MailSender = mail_sender.MailSender

type ShellExecutor = shell_executor.ShellExecutor

type TeamsSDK = teams_sdk.TeamsUtils

type GithubReleaseSDK = github_releases_sdk.ReleaseFactory

type Logger = zap.Logger
