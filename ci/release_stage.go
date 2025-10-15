package main

import (
	"context"
	"fmt"
	github_releases "github.com/rridane/golang-pipewire/modules/github_releases_sdk"
	"github.com/rridane/golang-pipewire/stage_layers"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"time"
)

type ReleaseStage struct{}

// Configure définit la commande et ses flags
func (s *ReleaseStage) Configure(cmd *cobra.Command) {
	cmd.Use = "release <version>"
	cmd.Short = "Crée et publie une nouvelle release GitHub"
	cmd.Long = `Crée une release GitHub pour la librairie, avec changelog et publication automatique sur pkg.go.dev.`

	cmd.Args = cobra.ExactArgs(1)
	cmd.Flags().String("branch", "main", "Branche cible du tag")
	cmd.Flags().Bool("prerelease", false, "Marquer la release comme pré-release")
	cmd.Flags().Bool("publish", true, "Publier la release (sinon, draft)")
	cmd.Flags().StringSlice("assets", nil, "Fichiers à uploader en assets")
}

func (s *ReleaseStage) Execute(
	ctx context.Context,
	cmd *cobra.Command,
	logger *zap.Logger,
	meter metric.Meter,
	tracer *trace.Tracer,
	releaseFactory *github_releases.ReleaseFactory,
) error {
	version := cmd.Flags().Args()[0]
	branch, _ := cmd.Flags().GetString("branch")
	isPrerelease, _ := cmd.Flags().GetBool("prerelease")
	shouldPublish, _ := cmd.Flags().GetBool("publish")
	assets, _ := cmd.Flags().GetStringSlice("assets")

	job := stage_layers.StartNewJobInstrumentation(ctx, "release", tracer, meter, logger)
	defer func() { job.End(nil) }()

	start := time.Now()
	logger.Info("🏷️ Démarrage du process de release", zap.String("version", version))

	// Étape 1 — Préparer la release
	release := releaseFactory.
		WithTag(version).
		WithReleaseName(fmt.Sprintf("Pipewire %s", version)).
		WithTargetBranch(branch).
		AsPrerelease(isPrerelease).
		AsDraft(!shouldPublish).
		WithBody("Automated release created via Pipewire CI").
		PrepareRelease()

	// Étape 2 — Création du tag Git
	release.CreateTag()

	// Étape 3 — Création de la release GitHub
	release.CreateRelease()

	// Étape 4 — Génération d’un changelog automatique
	release.WithChangelog()

	// Étape 5 — Upload d’assets éventuels
	if len(assets) > 0 {
		logger.Info("📦 Upload des assets", zap.Int("count", len(assets)))
		release.UploadAssets(assets)
	}

	// Étape 6 — Publication
	if shouldPublish {
		logger.Info("🚀 Publication de la release")
		release.Publish()
	}

	duration := time.Since(start)
	logger.Info("✅ Release terminée avec succès",
		zap.String("version", version),
		zap.String("duration", duration.String()),
	)
	job.End(nil)

	return nil
}
