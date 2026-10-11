package main

import (
	"context"
	"fmt"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"go.uber.org/zap"
)

// LintStage needs a reachable BuildKit daemon, which is why the GitHub
// workflow does not call it yet.
type LintStage struct{}

func (s *LintStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("lint")
	spec.SetDescription("Run golangci-lint in a container")
	spec.AddFlag("image", ru.StringFlag, "linter image", "golangci/golangci-lint:v1.62-alpine", false)
	return spec
}

func (s *LintStage) Execute(ctx context.Context, args ru.ParsedArgs, cb *mod.ContainerBuilder, log *zap.Logger) error {
	recipe := mod.Dockerfile{
		Args: map[string]any{"Image": args.String("image")},
		Template: `
			FROM {{.Image}} AS lint
			WORKDIR /src
			COPY . /src
			RUN --mount=type=cache,target=/go/pkg/mod \
			    --mount=type=cache,target=/root/.cache/go-build \
			    golangci-lint run ./...
		`,
	}

	if _, err := cb.Build(ctx, mod.BuildRequest{Dockerfile: recipe, ContextDir: "."}); err != nil {
		return fmt.Errorf("golangci-lint: %w", err)
	}

	log.Info("linter found nothing")
	return nil
}
