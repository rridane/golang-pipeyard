package python

import (
	"cmp"
	"context"
	"fmt"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"github.com/rridane/golang-pipeyard/x/stack"
	"go.uber.org/zap"
)

// pyrightNodeCache is where pyright puts the Node it downloads on first use.
const pyrightNodeCache = "/root/.cache/pyright-python"

// TypeCheckStage runs pyright over the repository, at the version the
// lockfile pins.
//
// It installs every extra, and that is not an optimisation: an import pyright
// cannot resolve is reported as an error against the code, so a stage that
// leaves the optional dependencies out invents faults. On a repository whose
// tests import its three extras, that was eight of them.
type TypeCheckStage struct {
	// OutDir is where the stage may write. Empty uses stack.DefaultOutDir.
	OutDir string

	// Toolchain decides the image, the uv version and what gets installed.
	// Its zero value installs every extra; the library pyright's own Node
	// needs is added whatever it says.
	Toolchain Toolchain

	// Paths are what to examine. Empty lets pyright read its own
	// configuration, which is where a repository already says it.
	Paths []string
}

func (s *TypeCheckStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("typecheck")
	spec.SetDescription("Check the Python types with pyright")
	return spec
}

func (s *TypeCheckStage) Execute(
	ctx context.Context,
	args ru.ParsedArgs,
	sh *mod.ShellExecutor,
	cb *mod.ContainerBuilder,
	log *zap.Logger,
) error {
	workspace, err := stack.NewWorkspace(ctx, sh, cmp.Or(s.OutDir, stack.DefaultOutDir))
	if err != nil {
		return err
	}

	result, err := s.toolchain().Tool("typecheck", "pyright "+stack.QuoteAll(s.Paths)).
		Run(ctx, cb, workspace)
	if err != nil {
		return err
	}

	if result.Output != "" {
		fmt.Println(result.Output)
	}
	if result.Failed() {
		return fmt.Errorf("pyright found type errors")
	}

	log.Info("the types hold", zap.Strings("paths", s.Paths))
	return nil
}

// toolchain adds what pyright needs and changes nothing the caller chose: the
// library its bundled Node links against, and somewhere to keep that Node
// between runs.
func (s *TypeCheckStage) toolchain() Toolchain {
	chain := s.Toolchain

	chain.Packages = append(chain.Packages, "libatomic1")
	chain.Caches = append(chain.Caches, pyrightNodeCache)

	return chain
}
