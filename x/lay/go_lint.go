package lay

import (
	"context"
	"fmt"

	"github.com/rridane/golang-pipeyard/x/mod"
	"go.uber.org/zap"
)

type GoLintLayer struct {
	LintImage string
}

func (l *GoLintLayer) Name() string { return "Go Lint" }

func (l *GoLintLayer) Run(ctx context.Context, logger *zap.Logger, cBuilder *mod.ContainerBuilder) error {

	logger.Info("🔹 preparing lint stage")

	if l.LintImage == "" {
		return fmt.Errorf("lint image is mandatory")
	}

	logger.Info(fmt.Sprintf("🔹 pulling container %s", l.LintImage))

	recette := mod.Dockerfile{
		Args: l,
		Template: `
			FROM {{.LintImage}} AS lint
			WORKDIR /src
			COPY . /src
			RUN golangci-lint run ./...
		`,
	}

	logger.Info("🔹 running golangci-lint")

	if _, err := cBuilder.Build(ctx, mod.BuildRequest{
		Dockerfile: recette,
		ContextDir: ".",
	}); err != nil {
		return fmt.Errorf("golangci-lint: %w", err)
	}

	logger.Info("✅ GoLintLayer completed successfully")
	return nil
}
