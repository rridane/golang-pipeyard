// Command ci is the pipeline that builds Pipeyard itself. Each stage is one
// invocation; the GitHub workflow keeps the sequencing.
package main

import (
	"context"
	"time"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/x/inj/container"
	"github.com/rridane/golang-pipeyard/x/inj/otel"
	"github.com/rridane/golang-pipeyard/x/inj/sys"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	pipe := pi.NewPipeline(pi.PipelineConfig{
		Name:             "pipeyard-ci",
		Version:          version,
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
		RegisterStage(&LintStage{}).
		RegisterStage(&ReleaseStage{})

	pipe.Run(context.Background())
}
