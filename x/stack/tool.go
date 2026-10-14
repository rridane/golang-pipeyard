package stack

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/rridane/golang-pipeyard/x/mod"
)

// reportPath is outside ContainerRoot, so the report cannot be mistaken for a
// repository file.
const reportPath = "/report/output"

// Tool is one command run in a pinned image over a Workspace.
type Tool struct {
	// Name labels the generated script and the errors.
	Name string

	// Image is the base image. Setup carries the Dockerfile lines that
	// install a tool with no image of its own.
	Image string
	Setup string

	// Script runs with ContainerRoot as its working directory.
	Script string

	// Caches are container paths BuildKit keeps between runs, mounted on the
	// step that runs the script and nowhere else.
	Caches []string
}

// Result is what the command printed and how it exited.
type Result struct {
	Output   string
	ExitCode int

	// Artifact is the image the run produced, for a stage that reads a file
	// back out of it.
	Artifact *mod.Artifact
}

// Failed reports a non-zero exit. On its own that is not an error: a linter
// and a scanner do not use their exit codes the same way, so the stage reads
// the output and decides.
func (r Result) Failed() bool { return r.ExitCode != 0 }

// recipeArgs is what the recipe below interpolates.
type recipeArgs struct{ Image, Setup, Script, Report, Mounts string }

// mounts renders one BuildKit cache mount per path.
func (t Tool) mounts() string {
	var b strings.Builder
	for _, target := range t.Caches {
		fmt.Fprintf(&b, " --mount=type=cache,target=%s", target)
	}
	return b.String()
}

// Run builds the image and hands back what the script printed. The RUN step
// always succeeds and records the exit code beside the output, because a
// failing BuildKit step keeps its own output out of reach.
func (t Tool) Run(ctx context.Context, cb *mod.ContainerBuilder, w Workspace) (Result, error) {
	script, err := w.Write(t.Name+".sh", []byte("set -u\n"+t.Script))
	if err != nil {
		return Result{}, fmt.Errorf("%s: write the script: %w", t.Name, err)
	}

	recipe := mod.Dockerfile{
		Args: recipeArgs{
			Image:  t.Image,
			Setup:  t.Setup,
			Script: script,
			Report: reportPath,
			Mounts: t.mounts(),
		},
		Template: `
			FROM {{.Image}}
			{{.Setup}}
			COPY . ` + ContainerRoot + `
			WORKDIR ` + ContainerRoot + `
			RUN{{.Mounts}} mkdir -p $(dirname {{.Report}}) \
			 && { sh {{.Script}} > {{.Report}} 2>&1; echo "exit=$?" >> {{.Report}}; }
		`,
	}

	artifact, err := cb.Build(ctx, mod.BuildRequest{Dockerfile: recipe, ContextDir: w.Dir})
	if err != nil {
		return Result{}, fmt.Errorf("%s: %w", t.Name, err)
	}

	raw, err := artifact.ReadFile(ctx, reportPath)
	if err != nil {
		return Result{}, fmt.Errorf("%s: read the report: %w", t.Name, err)
	}

	result, err := parseReport(raw)
	if err != nil {
		return Result{}, fmt.Errorf("%s: %w", t.Name, err)
	}
	result.Artifact = artifact

	return result, nil
}

// parseReport splits the output from the exit code the RUN step appended.
func parseReport(raw []byte) (Result, error) {
	raw = bytes.TrimRight(raw, "\n")

	cut := bytes.LastIndexByte(raw, '\n')
	last := string(raw[cut+1:])

	code, err := strconv.Atoi(strings.TrimPrefix(last, "exit="))
	if err != nil {
		return Result{}, fmt.Errorf("no exit code at the end of the output: %q", last)
	}

	output := ""
	if cut >= 0 {
		output = string(raw[:cut])
	}
	return Result{Output: output, ExitCode: code}, nil
}
