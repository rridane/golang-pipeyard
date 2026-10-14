package golang

import (
	"context"
	"fmt"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"go.uber.org/zap"
)

// BuildStage compiles the repository. It is worth running even where the tests
// already compile what they cover: a package nothing imports is only ever
// compiled here.
type BuildStage struct {
	// Packages is what --pkg falls back to. Empty uses DefaultPackages.
	Packages string
}

func (s *BuildStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("build")
	spec.SetDescription("Compile every package")
	spec.AddFlag("pkg", ru.StringFlag, "packages to compile", s.packages(), false)
	return spec
}

func (s *BuildStage) Execute(
	ctx context.Context,
	args ru.ParsedArgs,
	sh *mod.ShellExecutor,
	log *zap.Logger,
) error {
	if err := sh.Run(ctx, s.command(args)); err != nil {
		return err
	}

	log.Info("every package compiles")
	return nil
}

// command is where the flags come together, kept out of Execute so that a
// test reaches it without compiling anything.
func (s *BuildStage) command(args ru.ParsedArgs) string {
	return fmt.Sprintf("go build %s", args.String("pkg"))
}

func (s *BuildStage) packages() string {
	if s.Packages == "" {
		return DefaultPackages
	}
	return s.Packages
}
