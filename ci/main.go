package main

import (
	"context"
	"github.com/joho/godotenv"
	pipeline_provider "github.com/rridane/golang-pipeyard"
	injectors "github.com/rridane/golang-pipeyard/injectors"
	"go.uber.org/zap/zapcore"
	"log"
	"os"
)

func main() {

	ctx := context.Background()

	err := godotenv.Overload(".env")

	if err != nil {
		log.Println("Aucun fichier .env trouvé, utilisation des variables système")
	}

	tracerInjector, telemetryLoggerInjector, meterInjector := getInjectors()

	pipe := pipeline_provider.New(pipeline_provider.PipelineConfig{
		Name:             "ci_pipeline",
		ShortDescription: "CI CLI – All stages",
		LongDescription:  "Pipeline cli for pipeyard library",
		Version:          "1.0.0",
		Injectors: []any{
			&injectors.ShellExecutorInjector{},
			&tracerInjector,
			&telemetryLoggerInjector,
			&meterInjector,
		},
	})

	pipe.
		RegisterStage("test", &TestSTage{}).
		RegisterStage("build", &BuildStage{}).
		RegisterStage("release", &ReleaseStage{})

	pipe.Run()

	tracerInjector.Shutdown(ctx)
	telemetryLoggerInjector.Shutdown(ctx)
	meterInjector.Shutdown(ctx)

}

func getInjectors() (injectors.TracerInjector, injectors.MeterInjector, injectors.TelemetryLoggerInjector) {

	tracerInjector := injectors.TracerInjector{
		ServiceName:              "pipeyard",
		OtelExporterOtlpEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		Insecure:                 true,
	}

	telemetryLoggerInjector := injectors.TelemetryLoggerInjector{
		ServiceName:              "pipeyard",
		EnableOtelExport:         true,
		OtelExporterOtlpEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		LogLevel:                 zapcore.InfoLevel,
		OtelInsecure:             true,
	}

	meterInjector := injectors.MeterInjector{
		ServiceName:              "pipeyard",
		OtelExporterOtlpEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		Insecure:                 true,
	}

	return tracerInjector, meterInjector, telemetryLoggerInjector

}
