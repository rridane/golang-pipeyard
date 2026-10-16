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

// ArchitectureStage checks the import contracts a repository writes for
// itself with import-linter: which layer may import which, and which
// third-party package the core must never reach.
//
// It is the architecture made executable. A boundary that lives only in a
// diagram is a boundary that drifts, and no generic tool knows where a
// repository decided to draw its own.
type ArchitectureStage struct {
	// OutDir is where the stage may write. Empty uses stack.DefaultOutDir.
	OutDir string

	// Toolchain decides the image, the uv version and what gets installed.
	// Its zero value installs every extra, which a contract naming a
	// third-party package needs in order to resolve it.
	Toolchain Toolchain

	// Config is an import-linter configuration file. Empty lets it find its
	// own, which is pyproject.toml for most repositories.
	Config string
}

func (s *ArchitectureStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("architecture")
	spec.SetDescription("Check the import contracts with import-linter")
	return spec
}

func (s *ArchitectureStage) Execute(
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

	result, err := s.Toolchain.Tool("architecture", s.command()).Run(ctx, cb, workspace)
	if err != nil {
		return err
	}

	// The report names the contract that broke and the imports that broke
	// it, which is the whole of what the author needs.
	if result.Output != "" {
		fmt.Println(result.Output)
	}
	if result.Failed() {
		return fmt.Errorf("an import contract is broken")
	}

	log.Info("every import contract holds")
	return nil
}

func (s *ArchitectureStage) command() string {
	if s.Config == "" {
		return "lint-imports"
	}
	return "lint-imports --config " + stack.Quote(s.Config)
}
