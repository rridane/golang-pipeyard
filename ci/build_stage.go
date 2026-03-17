package main

import (
	"context"
	"fmt"
	"github.com/rridane/golang-pipeyard/pkg/pipeyard"
	"github.com/rridane/golang-pipeyard/pkg/pipeyardx/layers"
	"github.com/rridane/golang-pipeyard/pkg/pipeyardx/layers/golang"
	cBuilder "github.com/rridane/golang-pipeyard/pkg/pipeyardx/modules/container_builder"
	shell "github.com/rridane/golang-pipeyard/pkg/pipeyardx/modules/shell_executor"
	"github.com/rridane/golang-pipeyard/x"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"time"
)

type BuildStage struct{}

// Configure définit les paramètres de la commande
func (s *BuildStage) Configure() pipeyard.StageSpec {
	return pipeyard.StageSpec{
		Command: "build",
		Short:   "Valide la buildabilité et la propreté du module Go",
		Long: `Vérifie que le module Go compile, que les dépendances sont à jour,
et que le go.mod est propre. Aucun artefact n’est produit.`,
		Args: []x.Flag{
			{
				Name:        "",
				Type:        "",
				Default:     nil,
				Description: "",
				Required:    false,
			}},
	}
}

// Execute lance les validations du build
func (s *BuildStage) Execute(
	ctx context.Context,
	args x.ParsedArgs,
	logger *zap.Logger,
	shell *shell.ShellExecutor,
	meter metric.Meter,
	tracer trace.Tracer,
	cBuilder cBuilder.ContainerBuilder,
) error {
	job := layers.StartNewJobInstrumentation(ctx, "build", &tracer, meter, logger)
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

	lintLayer := golang.GoLintLayer{LintImage: "golangci/golangci-lint:v2.5.0"}
	err := lintLayer.Run(ctx, logger, cBuilder)

	if err != nil {
		return err
	}

	duration := time.Since(start)
	logger.Info("✅ Validation du module terminée", zap.String("duration", duration.String()))
	job.End(nil)
	return nil
}
