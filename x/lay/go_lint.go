package lay

import (
	"context"
	"fmt"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/x/mod"
	"go.uber.org/zap"
)

type GoLintLayer struct {
	CBuilder *mod.ContainerBuilder
	Logger   *zap.Logger
}

type GoLintLayerContext struct {
	LintImage  string
	ContextDir string
	Packages   string
}

func (l *GoLintLayer) Name() string { return "GoLint" }

func (l *GoLintLayer) Run(ctx context.Context, pipeyard pi.Ctx, c GoLintLayerContext) error {

	if l.CBuilder == nil {
		return fmt.Errorf("GoLintLayer: CBuilder is required")
	}
	if c.LintImage == "" {
		return fmt.Errorf("GoLintLayer: LintImage is required")
	}

	l.logger().Info("running golangci-lint", zap.String("image", c.LintImage))

	if _, err := l.CBuilder.Build(ctx, mod.BuildRequest{
		Dockerfile: c.recipe(),
		ContextDir: c.contextDir(pipeyard),
	}); err != nil {
		return fmt.Errorf("golangci-lint: %w", err)
	}

	l.logger().Info("linter found nothing")
	return nil
}

// The Go caches are mounted so a second run neither downloads nor recompiles
// the dependencies.
func (c GoLintLayerContext) recipe() mod.Dockerfile {
	return mod.Dockerfile{
		Args: c,
		Template: `
			FROM {{.LintImage}} AS lint
			WORKDIR /src
			COPY . /src
			RUN --mount=type=cache,target=/go/pkg/mod \
			    --mount=type=cache,target=/root/.cache/go-build \
			    golangci-lint run {{.Target}}
		`,
	}
}

func (c GoLintLayerContext) Target() string {
	if c.Packages != "" {
		return c.Packages
	}
	return "./..."
}

func (c GoLintLayerContext) contextDir(pipeyard pi.Ctx) string {
	if c.ContextDir != "" && c.ContextDir != "./" {
		return c.ContextDir
	}
	if pipeyard.CurrentDir != "" {
		return pipeyard.CurrentDir
	}
	return "."
}

func (l *GoLintLayer) logger() *zap.Logger {
	if l.Logger != nil {
		return l.Logger
	}
	return zap.NewNop()
}

// Both layers honour the contract now.
var (
	_ Layer[GoLintLayerContext]       = (*GoLintLayer)(nil)
	_ Layer[BuildAndPushLayerContext] = (*BuildAndPushLayer)(nil)
)
