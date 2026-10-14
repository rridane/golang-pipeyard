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

// LockStage checks that every lockfile still agrees with the manifest beside
// it. A drifted lockfile is the gap between what a repository says it installs
// and what it installs, and it only shows up on the machine that resolves
// next.
type LockStage struct {
	// OutDir is where the stage may write. Empty uses stack.DefaultOutDir.
	OutDir string

	// Projects are the directories holding a pyproject.toml and its
	// lockfile, relative to the repository root. Empty checks the
	// repository root.
	Projects []string

	// UvVersion is installed into Image. Empty uses DefaultUvVersion.
	UvVersion string

	// Image is where uv is installed. Empty uses DefaultImage.
	Image string
}

func (s *LockStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("lock-check")
	spec.SetDescription("Check that every lockfile still matches its manifest")
	return spec
}

func (s *LockStage) Execute(
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

	projects := s.projects()

	result, err := stack.Tool{
		Name:   "lock-check",
		Image:  s.image(),
		Setup:  "RUN pip install --no-cache-dir uv==" + s.uvVersion(),
		Script: s.script(projects),
	}.Run(ctx, cb, workspace)
	if err != nil {
		return err
	}

	if result.Output != "" {
		fmt.Println(result.Output)
	}
	if result.Failed() {
		return fmt.Errorf("a lockfile no longer matches its manifest, among %s",
			strings.Join(projects, ", "))
	}

	log.Info("every lockfile matches its manifest", zap.Strings("projects", projects))
	return nil
}

// script checks each project in turn, with the network left open on purpose: a
// manifest that gained a requirement can only be answered by resolving it, and
// refused the index uv reports a resolution failure instead of the drift it
// found.
func (s *LockStage) script(projects []string) string {
	script := new(stack.Script)
	for _, project := range projects {
		script.Step("uv lock --check --directory %s", stack.Quote(project))
	}
	return script.String()
}

func (s *LockStage) projects() []string {
	if len(s.Projects) == 0 {
		return []string{"."}
	}
	return s.Projects
}

func (s *LockStage) outDir() string {
	if s.OutDir == "" {
		return stack.DefaultOutDir
	}
	return s.OutDir
}

func (s *LockStage) uvVersion() string {
	if s.UvVersion == "" {
		return DefaultUvVersion
	}
	return s.UvVersion
}

func (s *LockStage) image() string {
	if s.Image == "" {
		return DefaultImage
	}
	return s.Image
}
