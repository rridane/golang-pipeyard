package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"go.uber.org/zap"
)

// releaseTag accepts a version written either way and always produces the tag
// Git carries, so --version 0.1.0 and --version v0.1.0 publish the same thing.
func releaseTag(version string) string {
	return "v" + strings.TrimPrefix(version, "v")
}

// PublishStage is separate from release so that a release without a GitHub
// token still runs: a local `release --version x.y.z` builds the image and
// stops there.
type PublishStage struct{}

func (s *PublishStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("publish")
	spec.SetDescription("Attach the binary to a GitHub release")
	spec.AddFlag("version", ru.StringFlag, "semver version, without a leading v", "", true)
	spec.AddFlag("asset", ru.StringFlag, "file to attach", "dist/pipeyard-ci", false)
	return spec
}

func (s *PublishStage) Execute(ctx context.Context, args ru.ParsedArgs,
	releases *mod.GithubReleaseSDK, log *zap.Logger) error {

	tag := releaseTag(args.String("version"))
	asset := args.String("asset")

	// Created as a draft, published last. A release becomes visible to
	// releases/latest/download the moment it leaves draft, so attaching the
	// binary first is what stops a run from fetching a release without one.
	published := releases.
		WithTag(tag).
		WithReleaseName(tag).
		PrepareRelease().
		CreateRelease().
		UploadAssets([]string{asset}).
		Publish()

	if err := published.Err(); err != nil {
		return fmt.Errorf("publish %s: %w", tag, err)
	}

	log.Info("release published", zap.String("tag", tag), zap.String("asset", asset))
	return nil
}
