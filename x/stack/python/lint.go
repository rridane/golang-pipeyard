package python

import (
	"context"
	"fmt"
	"strings"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"github.com/rridane/golang-pipeyard/x/stack"
	"go.uber.org/zap"
)

// LintStage runs ruff over the checkout: the checks, then the formatting. Both
// run even when the first fails, so one invocation shows everything to fix.
type LintStage struct {
	// OutDir is where the stage may write. Empty uses stack.DefaultOutDir.
	OutDir string

	// Paths are what to lint, relative to the repository root. Empty lints
	// the whole repository.
	Paths []string

	// RuffVersion is installed into Image. Empty uses DefaultRuffVersion.
	RuffVersion string

	// Image is where ruff is installed. Empty uses DefaultImage.
	Image string

	// SkipFormat leaves the formatting check out, for a repository not
	// formatted yet. It is meant to be removed.
	SkipFormat bool
}

func (s *LintStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("lint")
	spec.SetDescription("Check the Python style and formatting with ruff")
	return spec
}

func (s *LintStage) Execute(
	ctx context.Context,
	args ru.ParsedArgs,
	sh *mod.ShellExecutor,
	cb *mod.ContainerBuilder,
	log *zap.Logger,
) error {
	workspace, err := stack.NewWorkspace(ctx, sh, s.outDir())
	if err != nil {
		return err
	}

	paths := s.paths()

	result, err := stack.Tool{
		Name:   "lint",
		Image:  s.image(),
		Setup:  "RUN pip install --no-cache-dir ruff==" + s.ruffVersion(),
		Script: s.script(paths),
	}.Run(ctx, cb, workspace)
	if err != nil {
		return err
	}

	if result.Output != "" {
		fmt.Println(result.Output)
	}
	if result.Failed() {
		return fmt.Errorf("ruff found something to fix in %s", strings.Join(paths, ", "))
	}

	log.Info("the Python is clean", zap.Strings("paths", paths))
	return nil
}

// script asks ruff not to cache: the cache would be written into the image
// layer and thrown away with it.
func (s *LintStage) script(paths []string) string {
	quoted := make([]string, 0, len(paths))
	for _, p := range paths {
		quoted = append(quoted, stack.Quote(p))
	}
	targets := strings.Join(quoted, " ")

	script := new(stack.Script).Step("ruff check --no-cache %s", targets)
	if !s.SkipFormat {
		script.Step("ruff format --check --no-cache %s", targets)
	}
	return script.String()
}

func (s *LintStage) paths() []string {
	if len(s.Paths) == 0 {
		return []string{"."}
	}
	return s.Paths
}

func (s *LintStage) outDir() string {
	if s.OutDir == "" {
		return stack.DefaultOutDir
	}
	return s.OutDir
}

func (s *LintStage) ruffVersion() string {
	if s.RuffVersion == "" {
		return DefaultRuffVersion
	}
	return s.RuffVersion
}

func (s *LintStage) image() string {
	if s.Image == "" {
		return DefaultImage
	}
	return s.Image
}
