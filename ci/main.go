// Command ci is the pipeline that builds Pipeyard itself.
//
// Each stage is one invocation. The GitHub workflow calls them in order; it
// keeps the sequencing, this binary does the work.
package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/inj/container"
	"github.com/rridane/golang-pipeyard/x/inj/otel"
	"github.com/rridane/golang-pipeyard/x/inj/sys"
	"github.com/rridane/golang-pipeyard/x/mod"
	"go.uber.org/zap"
)

/* ───────────────────────── fmt ───────────────────────── */

// FormatStage fails when a file is not gofmt'd, and names the files.
type FormatStage struct{}

func (s *FormatStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("fmt")
	spec.SetDescription("Check that every package is gofmt'd")
	return spec
}

func (s *FormatStage) Execute(ctx context.Context, args ru.ParsedArgs, sh *mod.ShellExecutor, log *zap.Logger) error {
	out, err := sh.Output(ctx, "gofmt -l pi ru x ci")
	if err != nil {
		return fmt.Errorf("gofmt: %w", err)
	}

	unformatted := strings.Fields(string(out))
	if len(unformatted) > 0 {
		return fmt.Errorf("%d file(s) need gofmt: %s", len(unformatted), strings.Join(unformatted, " "))
	}

	log.Info("every package is formatted")
	return nil
}

/* ───────────────────────── test ───────────────────────── */

type TestStage struct{}

func (s *TestStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("test")
	spec.SetDescription("Run the test suite")
	spec.AddFlag("pkg", ru.StringFlag, "packages to test", "./...", false)
	spec.AddFlag("race", ru.BoolFlag, "enable the race detector", true, false)
	spec.AddFlag("timeout", ru.DurationFlag, "test timeout", "5m", false)
	return spec
}

func (s *TestStage) Execute(ctx context.Context, args ru.ParsedArgs, sh *mod.ShellExecutor, log *zap.Logger) error {
	flags := fmt.Sprintf("-timeout %s", args.Duration("timeout"))
	if args.Bool("race") {
		flags += " -race"
	}

	log.Info("running tests", zap.String("pkg", args.String("pkg")), zap.Bool("race", args.Bool("race")))

	return sh.Run(ctx, fmt.Sprintf("go test %s %s", flags, args.String("pkg")))
}

/* ───────────────────────── vet ───────────────────────── */

type VetStage struct{}

func (s *VetStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("vet")
	spec.SetDescription("Run go vet on every package")
	return spec
}

func (s *VetStage) Execute(ctx context.Context, args ru.ParsedArgs, sh *mod.ShellExecutor, log *zap.Logger) error {
	if err := sh.Run(ctx, "go vet ./..."); err != nil {
		return err
	}
	log.Info("vet found nothing")
	return nil
}

/* ───────────────────────── lint ───────────────────────── */

// LintStage runs golangci-lint in a container, so no one has to install it.
// It needs a reachable BuildKit daemon, which is why the GitHub workflow does
// not call it yet.
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

func main() {
	pipe := pi.NewPipeline(pi.PipelineConfig{
		Name:             "ci",
		Version:          "0.1.0",
		ShortDescription: "Pipeyard's own pipeline",
		LongDescription: "The pipeline that builds Pipeyard, written with Pipeyard.\n" +
			"Each command is one stage; the CI engine calls them in order.",
		Injectors: []any{
			&otel.TelemetryLoggerInjector{ServiceName: "pipeyard-ci"},
			&sys.ShellExecutorInjector{Timeout: 15 * time.Minute},
			&container.ContainerBuilderInjector{},
		},
	})

	pipe.RegisterStage(&FormatStage{}).
		RegisterStage(&VetStage{}).
		RegisterStage(&TestStage{}).
		RegisterStage(&LintStage{})

	pipe.Run(context.Background())
}
