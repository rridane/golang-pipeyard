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

	logger.Info("🔹 Préparation du stage de lint")

	if l.LintImage == "" {
		return fmt.Errorf("lint image is mandatory")
	}

	logger.Info(fmt.Sprintf("🔹 Pulling container %s", l.LintImage))

	container := cBuilder.InitContainer(mod.ContainerSpec{
		Image:      l.LintImage,
		Env:        map[string]string{"FOO": "bar"},
		WorkDir:    "/src",
		EntryPoint: "sh",
		Logger:     logger,
	})

	logger.Info("🔹 Execute golangci-lint")
	container.ExecShell("mkdir src").CopyFromHost("./", "./src").ExecShell("cd src && golangci-lint run ./...").Apply()

	logger.Info("✅ GoLintLayer layer terminé avec succès")
	return nil
}
