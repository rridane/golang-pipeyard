package main

import (
	"context"
	container_builder "github.com/rridane/golang-pipeyard/modules/container_builder"
	"github.com/rridane/golang-pipeyard/stage_layers"
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

	job := stage_layers.StartNewJobInstrumentation(ctx, "test", &tracer, meter, logger)

	timeout, _ := cmd.Flags().GetDuration("timeout")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	//err := shell.Run(ctx, "cd .. && go test -v cover ./...")

	spec := container_builder.ContainerSpec{
		Image:      "golang:1.24.8",
		Env:        map[string]string{"FOO": "bar"},
		WorkDir:    "/src",
		EntryPoint: "sh",
		Logger:     logger,
	}

	c := cBuilder.InitContainer(spec)

	c.Exec("go test -v -cover ./...")

	//job.End(err)
	job.End(nil)

	return nil
}
