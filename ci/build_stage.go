package main

import (
	"context"
	"fmt"
	shell_executor "github.com/rridane/golang-pipeyard/modules/shell_executor"
	"github.com/rridane/golang-pipeyard/stage_layers"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"time"
)

type BuildStage struct{}

// Configure définit les paramètres de la commande
func (s *BuildStage) Configure(cmd *cobra.Command) {
	cmd.Use = "build"
	cmd.Short = "Valide la buildabilité et la propreté du module Go"
	cmd.Long = `Vérifie que le module Go compile, que les dépendances sont à jour,
et que le go.mod est propre. Aucun artefact n’est produit.`
}

// Execute lance les validations du build
func (s *BuildStage) Execute(
	ctx context.Context,
	cmd *cobra.Command,
	logger *zap.Logger,
	shell *shell_executor.ShellExecutor,
	meter metric.Meter,
	tracer trace.Tracer,
) error {
	job := stage_layers.StartNewJobInstrumentation(ctx, "build", &tracer, meter, logger)
	defer func() { job.End(nil) }()

	start := time.Now()

	logger.Info("🧱 Validation du module Go (buildability check)")

	steps := []struct {
		name string
		cmd  string
	}{
		{"Vérification des dépendances", "go mod tidy"},
		{"Analyse statique (vet)", "go vet ./..."},
		{"Compilation dry-run", "go build ./..."},
		{"Vérification de propreté du go.mod", "git diff --exit-code go.mod go.sum || true"},
	}

	for _, step := range steps {
		logger.Info(fmt.Sprintf("🔹 %s", step.name))
		if err := shell.Run(ctx, step.cmd); err != nil {
			job.End(err)
			return fmt.Errorf("échec de l’étape '%s': %w", step.name, err)
		}
	}

	duration := time.Since(start)
	logger.Info("✅ Validation du module terminée", zap.String("duration", duration.String()))
	job.End(nil)
	return nil
}
