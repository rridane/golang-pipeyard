package python

import (
	"cmp"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"github.com/rridane/golang-pipeyard/x/stack"
	"go.uber.org/zap"
)

// noTestsCollected is what pytest exits with when it found nothing to run. It
// is a success as far as the shell is concerned, which is why this stage
// treats it apart: a suite that runs no test is the one result a pipeline
// must never report as green.
const noTestsCollected = 5

// PytestStage runs the suite at the versions the lockfile pins.
type PytestStage struct {
	// OutDir is where the stage may write. Empty uses stack.DefaultOutDir.
	OutDir string

	// Toolchain decides the image, the uv version and what gets installed.
	// Its zero value installs every extra, because a test suite is the part
	// of a repository that imports the most.
	Toolchain Toolchain

	// Paths are what to collect. Empty lets pytest read its own
	// configuration.
	Paths []string

	// Markers selects by marker, as pytest -m does: "not ray" leaves out
	// what only runs inside a Ray job.
	Markers string

	// Env is configuration the suite reads, written into the script.
	//
	// It is not a place for a secret: the script lives in the build context
	// and the values reach the image. A secret belongs in a mount, which
	// this stage does not offer yet.
	Env map[string]string

	// AllowNoTests accepts an empty collection. It is for a repository whose
	// suite is still to be written, and it is meant to be removed.
	AllowNoTests bool
}

func (s *PytestStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("test")
	spec.SetDescription("Run the test suite with pytest")
	spec.AddFlag("markers", ru.StringFlag, "select tests by marker, as pytest -m does", s.Markers, false)
	return spec
}

func (s *PytestStage) Execute(
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

	chain := s.Toolchain
	chain.AllExtras = true

	tool := chain.Tool("test", s.command(args.String("markers")))
	tool.Script = s.exports() + tool.Script

	result, err := tool.Run(ctx, cb, workspace)
	if err != nil {
		return err
	}

	if result.Output != "" {
		fmt.Println(result.Output)
	}

	switch {
	case result.ExitCode == noTestsCollected && !s.AllowNoTests:
		return fmt.Errorf("pytest collected no test at all")
	case result.ExitCode == noTestsCollected:
		log.Warn("pytest collected no test, accepted")
		return nil
	case result.Failed():
		return fmt.Errorf("the suite failed")
	}

	log.Info("the suite passed", zap.String("markers", args.String("markers")))
	return nil
}

func (s *PytestStage) command(markers string) string {
	command := "pytest"

	if markers != "" {
		command += " -m " + stack.Quote(markers)
	}
	if len(s.Paths) > 0 {
		command += " " + stack.QuoteAll(s.Paths)
	}
	return command
}

// exports writes the configuration ahead of the command, sorted so that two
// runs produce the same script and BuildKit can reuse its cache.
func (s *PytestStage) exports() string {
	names := make([]string, 0, len(s.Env))
	for name := range s.Env {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	for _, name := range names {
		fmt.Fprintf(&b, "export %s=%s\n", name, stack.Quote(s.Env[name]))
	}
	return b.String()
}
