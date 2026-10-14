package golang

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

// LintStage runs golangci-lint in a container, over the files git would see. A
// linter is only useful while everyone runs the same version, and the image is
// what holds everyone to it.
type LintStage struct {
	// OutDir is where the stage may write. Empty uses stack.DefaultOutDir.
	OutDir string

	// Image is the linter, pinned. Empty uses DefaultLinterImage. A field
	// rather than a flag: which linter a repository answers to is not a
	// choice to make at the command line, where runs could disagree.
	Image string

	// Packages is what to examine. Empty examines DefaultPackages.
	Packages string
}

func (s *LintStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("lint")
	spec.SetDescription("Run golangci-lint in a container")
	return spec
}

func (s *LintStage) Execute(
	ctx context.Context,
	args ru.ParsedArgs,
	sh *mod.ShellExecutor,
	cb *mod.ContainerBuilder,
	log *zap.Logger,
) error {
	image := cmp.Or(s.Image, DefaultLinterImage)
	packages := cmp.Or(s.Packages, DefaultPackages)

	workspace, err := stack.NewWorkspace(ctx, sh, cmp.Or(s.OutDir, stack.DefaultOutDir))
	if err != nil {
		return err
	}

	result, err := stack.Tool{
		Name:   "lint",
		Image:  image,
		Script: "golangci-lint run " + packages + "\n",
		// The paths come from `go env` and `golangci-lint cache status` inside
		// the image: a cache mounted anywhere else is never read, and nothing
		// says so. The last one holds the analysis and pays most.
		Caches: []string{
			"/go/pkg/mod",
			"/root/.cache/go-build",
			"/root/.cache/golangci-lint",
		},
	}.Run(ctx, cb, workspace)
	if err != nil {
		return err
	}

	if result.Output != "" {
		fmt.Println(result.Output)
	}
	if result.Failed() {
		return fmt.Errorf("the linter found something in %s", packages)
	}

	log.Info("linter found nothing", zap.String("image", image))
	return nil
}
