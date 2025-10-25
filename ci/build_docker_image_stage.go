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

type BuildDockerStage struct {
}

func (s *BuildDockerStage) Configure(cmd *cobra.Command) {

	cmd.Use = "buildDockerImage"
	cmd.Short = "Build image docker et push"

	cmd.Flags().Duration("timeout", 30*time.Second, "Timeout en secondes")

}

func (s *BuildDockerStage) Execute(ctx context.Context, cmd *cobra.Command, logger *zap.Logger, meter metric.Meter, tracer trace.Tracer, cBuilder *container_builder.ContainerBuilder) error {

	job := stage_layers.StartNewJobInstrumentation(ctx, "docker build", &tracer, meter, logger)

	timeout, _ := cmd.Flags().GetDuration("timeout")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	spec := container_builder.ContainerSpec{
		DockerfilePath: "./ci/Dockerfile",
		ContextDir:     "./",
		Env:            map[string]string{"FOO": "bar"},
		WorkDir:        "/home",
		EntryPoint:     "sh",
		Logger:         logger,
	}

	c := cBuilder.
		InitContainer(spec).
		Exec("curl https://nexus.fr > google_fr_txt").CopyToHost("google_fr_txt", "./").Apply()

	err := c.Push(ctx, "rridane/pipeyard:0.0.1", false)

	if err != nil {
		logger.Error(err.Error())
		return err
	}

	//job.End(err)
	job.End(nil)

	return nil
}
