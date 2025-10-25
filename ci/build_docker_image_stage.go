package main

import (
	"context"
	"time"

	container_builder "github.com/rridane/golang-pipeyard/modules/container_builder"
	"github.com/rridane/golang-pipeyard/stage_layers"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type BuildDockerStage struct{}

func (s *BuildDockerStage) Configure(cmd *cobra.Command) {
	cmd.Use = "buildDockerImage"
	cmd.Short = "Construit et push une image Docker via BuildKit"

	cmd.Flags().Duration("timeout", 300*time.Second, "Durée maximale d’exécution du build Docker")
	cmd.Flags().String("image", "rridane/pipeyard:0.0.1", "Nom complet de l’image à builder et push")
	cmd.Flags().String("context", "./", "Répertoire de contexte du Dockerfile")
	cmd.Flags().String("dockerfile", "./ci/Dockerfile", "Chemin du Dockerfile à utiliser")
	cmd.Flags().Bool("push", true, "Activer le push de l’image vers le registry")
}

func (s *BuildDockerStage) Execute(
	ctx context.Context,
	cmd *cobra.Command,
	logger *zap.Logger,
	meter metric.Meter,
	tracer trace.Tracer,
	cBuilder *container_builder.ContainerBuilder,
) error {

	job := stage_layers.StartNewJobInstrumentation(ctx, "docker build", &tracer, meter, logger)

	// Récupération des flags CLI
	timeout, _ := cmd.Flags().GetDuration("timeout")
	imageName, _ := cmd.Flags().GetString("image")
	contextDir, _ := cmd.Flags().GetString("context")
	dockerfile, _ := cmd.Flags().GetString("dockerfile")
	pushEnabled, _ := cmd.Flags().GetBool("push")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Construction du container spec
	spec := container_builder.ContainerSpec{
		DockerfilePath: dockerfile,
		ContextDir:     contextDir,
		Env:            map[string]string{"FOO": "bar"},
		WorkDir:        "/home",
		EntryPoint:     "sh",
		Logger:         logger,
	}

	// Construction du conteneur BuildKit
	container := cBuilder.
		InitContainer(spec).
		Apply()

	if pushEnabled {
		if err := container.Push(ctx, imageName, false); err != nil {
			logger.Error("Échec du push de l’image", zap.Error(err))
			job.End(err)
			return err
		}
		logger.Info("Image poussée avec succès", zap.String("image", imageName))
	} else {
		logger.Info("Push désactivé — image non envoyée", zap.String("image", imageName))
	}

	job.End(nil)
	return nil
}
