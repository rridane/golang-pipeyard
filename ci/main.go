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
	"github.com/rridane/golang-pipeyard/x/inj/forge"
	"github.com/rridane/golang-pipeyard/x/inj/otel"
	"github.com/rridane/golang-pipeyard/x/inj/sys"
	"github.com/rridane/golang-pipeyard/x/lay"
	"github.com/rridane/golang-pipeyard/x/mod"
	"github.com/rridane/golang-pipeyard/x/stack/golang"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

const serviceName = "pipeyard-ci"

func main() {
	loadEnv()

	endpoint, insecure := otlpTarget()
	protocol := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL")
	owner, repo := splitProject(pi.ReadIdentity().Project)

	tracer := &otel.TracerInjector{
		ServiceName:              serviceName,
		OtelExporterOtlpEndpoint: endpoint,
		Insecure:                 insecure,
		Protocol:                 protocol,
	}
	meter := &otel.MeterInjector{
		ServiceName:              serviceName,
		OtelExporterOtlpEndpoint: endpoint,
		Insecure:                 insecure,
		Protocol:                 protocol,
	}
	logger := &otel.TelemetryLoggerInjector{
		ServiceName:              serviceName,
		EnableOtelExport:         endpoint != "",
		OtelExporterOtlpEndpoint: endpoint,
		OtelInsecure:             insecure,
		Protocol:                 protocol,
	}

	pipe := pi.NewPipeline(pi.PipelineConfig{
		Name:             serviceName,
		Version:          version,
		ShortDescription: "Pipeyard's own pipeline",
		LongDescription: "The pipeline that builds Pipeyard, written with Pipeyard.\n" +
			"Each command is one stage; the CI engine calls them in order.",

		Injectors: []any{
			logger,
			&sys.IdentityInjector{},
			&sys.ShellExecutorInjector{Timeout: 15 * time.Minute},
			&container.ContainerBuilderInjector{Registries: registryFromEnv()},
			&forge.GitHubReleaseInjector{
				Token: os.Getenv("GITHUB_TOKEN"),
				Owner: owner,
				Repo:  repo,
			},
		},

		Observer: &lay.StageObserver{
			Tracer: tracer.Inject(serviceName),
			Meter:  meter.Inject(serviceName),
			Logger: logger.Inject(""),
			Flags:  []string{"push"},
			Shutdown: []func(context.Context){
				tracer.Shutdown, meter.Shutdown, logger.Shutdown,
			},
		},
	})

	// The five Go stages come from the stack, configured and not copied:
	// everything this repository needs to say about them fits in a field.
	// release and publish stay here, because what they do is this project's
	// own and there is no field that would express it.
	pipe.RegisterStage(&golang.BuildStage{}).
		RegisterStage(&golang.FormatStage{}).
		RegisterStage(&golang.VetStage{}).
		RegisterStage(&golang.TestStage{}).
		RegisterStage(&golang.LintStage{}).
		RegisterStage(&ReleaseStage{}).
		RegisterStage(&PublishStage{})

	pipe.Run(context.Background())
}

// loadEnv reads a .env when there is one, so a laptop run picks up the same
// settings the CI gets from its own environment. A missing file is normal.
func loadEnv() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "warning: .env not loaded: %v\n", err)
	}
}

// registryFromEnv reads the one registry this run pushes to. From the
// environment rather than a flag: a token on the command line ends up in the
// runner's logs and in ps. An empty host means Docker Hub.
func registryFromEnv() map[string]mod.RegistryCredential {
	user := os.Getenv("PIPEYARD_REGISTRY_USER")
	token := os.Getenv("PIPEYARD_REGISTRY_TOKEN")

	if user == "" || token == "" {
		return nil
	}

	return map[string]mod.RegistryCredential{
		os.Getenv("PIPEYARD_REGISTRY_HOST"): {Username: user, Password: token},
	}
}

// splitProject cuts owner/repo apart. The project name is written the forge's
// way, so reading it that way belongs here, in the pipeline that knows its
// forge, and not in the framework.
func splitProject(project string) (owner, repo string) {
	owner, repo, _ = strings.Cut(project, "/")
	return owner, repo
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
