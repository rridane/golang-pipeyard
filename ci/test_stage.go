package main

import (
	"context"
	shell_executor "github.com/rridane/golang-pipewire/modules/shell_executor"
	"github.com/rridane/golang-pipewire/stage_layers"
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

func (s *TestSTage) Execute(ctx context.Context, cmd *cobra.Command, logger *zap.Logger, shell *shell_executor.ShellExecutor, meter metric.Meter, tracer *trace.Tracer) error {

	job := stage_layers.StartNewJobInstrumentation(ctx, "test", tracer, meter, logger)

	timeout, _ := cmd.Flags().GetDuration("timeout")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := shell.Run(ctx, "cd .. && go test -v cover ./...")

	job.End(err)

	return err
}
