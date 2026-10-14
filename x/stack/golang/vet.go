package golang

import (
	"cmp"
	"context"
	"fmt"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"go.uber.org/zap"
)

// VetStage runs the checks the compiler leaves out: a format verb that does
// not match its argument, a lock copied by value, a struct tag nothing reads.
type VetStage struct {
	// Packages is what --pkg falls back to. Empty uses DefaultPackages.
	Packages string
}

func (s *VetStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("vet")
	spec.SetDescription("Run go vet on every package")
	spec.AddFlag("pkg", ru.StringFlag, "packages to examine", cmp.Or(s.Packages, DefaultPackages), false)
	return spec
}

func (s *VetStage) Execute(
	ctx context.Context,
	args ru.ParsedArgs,
	sh *mod.ShellExecutor,
	log *zap.Logger,
) error {
	if err := sh.Run(ctx, s.command(args)); err != nil {
		return err
	}

	log.Info("found nothing")
	return nil
}

// command is where the flags come together, kept out of Execute so that a
// test reaches it without running vet.
func (s *VetStage) command(args ru.ParsedArgs) string {
	return fmt.Sprintf("go vet %s", args.String("pkg"))
}
