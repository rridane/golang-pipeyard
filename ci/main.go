// Command ci is the pipeline that builds Pipeyard itself. Each stage is one
// invocation; the GitHub workflow keeps the sequencing.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/x/inj/container"
	"github.com/rridane/golang-pipeyard/x/inj/otel"
	"github.com/rridane/golang-pipeyard/x/inj/sys"
	"github.com/rridane/golang-pipeyard/x/lay"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

const serviceName = "pipeyard-ci"

func main() {
	loadEnv()

	endpoint, insecure := otlpTarget()

	tracer := &otel.TracerInjector{
		ServiceName:              serviceName,
		OtelExporterOtlpEndpoint: endpoint,
		Insecure:                 insecure,
	}
	meter := &otel.MeterInjector{
		ServiceName:              serviceName,
		OtelExporterOtlpEndpoint: endpoint,
		Insecure:                 insecure,
	}
	logger := &otel.TelemetryLoggerInjector{
		ServiceName:              serviceName,
		EnableOtelExport:         endpoint != "",
		OtelExporterOtlpEndpoint: endpoint,
		OtelInsecure:             insecure,
	}

	pipe := pi.NewPipeline(pi.PipelineConfig{
		Name:             serviceName,
		Version:          version,
		ShortDescription: "Pipeyard's own pipeline",
		LongDescription: "The pipeline that builds Pipeyard, written with Pipeyard.\n" +
			"Each command is one stage; the CI engine calls them in order.",

		Injectors: []any{
			logger,
			&sys.ShellExecutorInjector{Timeout: 15 * time.Minute},
			&container.ContainerBuilderInjector{},
		},

		Observer: &lay.StageObserver{
			Tracer: tracer.Inject(serviceName),
			Meter:  meter.Inject(serviceName),
			Logger: logger.Inject("observer"),
			Flags:  []string{"push"},
			Shutdown: []func(context.Context){
				tracer.Shutdown, meter.Shutdown, logger.Shutdown,
			},
		},
	})

	pipe.RegisterStage(&FormatStage{}).
		RegisterStage(&VetStage{}).
		RegisterStage(&TestStage{}).
		RegisterStage(&LintStage{}).
		RegisterStage(&ReleaseStage{})

	pipe.Run(context.Background())
}

// loadEnv reads a .env when there is one, so a laptop run picks up the same
// settings the CI gets from its own environment. A missing file is normal.
func loadEnv() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "warning: .env not loaded: %v\n", err)
	}
}

// otlpTarget reads the standard variable, which is a URL, and hands back the
// host:port the gRPC exporters take plus the transport the scheme asked for.
// Reading the scheme here keeps the decision explicit instead of leaning on
// the SDK's own environment precedence.
func otlpTarget() (endpoint string, insecure bool) {
	raw := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	insecure = strings.HasPrefix(raw, "http://")

	raw = strings.TrimPrefix(raw, "http://")
	raw = strings.TrimPrefix(raw, "https://")

	return strings.TrimSuffix(raw, "/"), insecure
}
