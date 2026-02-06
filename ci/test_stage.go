package main

import (
	"context"
	"github.com/rridane/golang-pipeyard/layers"
	container_builder "github.com/rridane/golang-pipeyard/pkg/container_builder"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"time"
)

type TestSTage struct {
}

func (s *TestSTage) Configure(cmd *cobra.Command) {

	cmd.Use = "test"
	cmd.Short = "Lance l'ensemble des tests et échoue si un test échoue"

	cmd.Flags().Duration("timeout", 30*time.Second, "Timeout en secondes")

}

func (s *TestSTage) Execute(ctx context.Context, cmd *cobra.Command, logger *zap.Logger, meter metric.Meter, tracer trace.Tracer, cBuilder *container_builder.ContainerBuilder) error {

	timeout, _ := cmd.Flags().GetDuration("timeout")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	job := layers.StartNewJobInstrumentation(ctx, "test", &tracer, meter, logger)

	spec := container_builder.ContainerSpec{
		Image:      "golang:1.24.9",
		Env:        map[string]string{"FOO": "bar"},
		WorkDir:    "/",
		EntryPoint: "sh",
		Logger:     logger,
	}

	c := cBuilder.InitContainer(spec)

	c.ExecShell("mkdir src").CopyFromHost("./", "./src").ExecShell("cd ./src && go test -v -cover ./...").Apply()

	job.End(nil)

	return nil
}
