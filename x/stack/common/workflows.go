package common

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"github.com/rridane/golang-pipeyard/x/stack"
	"go.uber.org/zap"
)

// workflowGlobs are the files the auditor reads. A repository with none of
// them has nothing to audit, which the auditor would report as a failure.
var workflowGlobs = []string{
	".github/workflows/*.yml", ".github/workflows/*.yaml",
	"action.yml", "action.yaml",
}

// WorkflowsStage audits the GitHub Actions workflows with zizmor: what a step
// can reach, what it leaves behind and what it trusts.
type WorkflowsStage struct {
	// OutDir is where the stage may write. Empty uses stack.DefaultOutDir.
	OutDir string

	// Policy is a zizmor configuration. Set, it replaces whatever the
	// repository carries, so the organisation decides which audits hold.
	// Empty runs every audit. An exemption is declared here and nowhere
	// else: the stage runs --no-ignores, so an "ignore" comment left in a
	// workflow does not apply.
	Policy string

	// Image is the auditor, pinned. Empty uses DefaultZizmorImage.
	Image string
}

// WorkflowFinding is one weakness at one place in a workflow.
type WorkflowFinding struct {
	Audit    string
	Severity string
	File     string
	Line     int
	Detail   string
}

func (f WorkflowFinding) String() string {
	return fmt.Sprintf("%s:%d %s (%s): %s", f.File, f.Line, f.Audit, f.Severity, f.Detail)
}

func (s *WorkflowsStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("workflows")
	spec.SetDescription("Audit the GitHub Actions workflows with zizmor")
	return spec
}

func (s *WorkflowsStage) Execute(
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

	if !hasWorkflows(workspace.Dir) {
		log.Info("no workflow to audit")
		return nil
	}

	policy := ""
	if s.Policy != "" {
		host, inContainer, err := workspace.Generated("zizmor.yml")
		if err != nil {
			return err
		}
		if err := os.WriteFile(host, []byte(s.Policy), 0o644); err != nil {
			return fmt.Errorf("write the policy: %w", err)
		}
		policy = inContainer
	}

	result, err := stack.Tool{
		Name:   "workflows",
		Image:  cmp.Or(s.Image, DefaultZizmorImage),
		Script: s.script(policy),
	}.Run(ctx, cb, workspace)
	if err != nil {
		return err
	}

	findings, err := parseWorkflowFindings([]byte(result.Output))
	if err != nil {
		return fmt.Errorf("%w (exit %d):\n%s", err, result.ExitCode, result.Output)
	}

	for _, f := range findings {
		log.Error("workflow finding", zap.String("finding", f.String()))
	}
	if len(findings) > 0 {
		return fmt.Errorf("%d finding(s) in the workflows", len(findings))
	}

	log.Info("the workflows hold")
	return nil
}

// script runs offline: the online audits need a token, and a stage that
// behaves differently with and without one is not reproducible. The exit code
// is left aside, the report says what was found. zizmor's own logging goes to
// stderr even with --no-progress, and the trailer the RUN step appends would
// land in the middle of it, so stderr is dropped.
func (s *WorkflowsStage) script(policy string) string {
	config := ""
	if policy != "" {
		config = "--config " + stack.Quote(policy) + " "
	}
	return fmt.Sprintf("zizmor --offline --no-progress --no-ignores --format json %s. 2>/dev/null\n", config)
}

func hasWorkflows(dir string) bool {
	for _, glob := range workflowGlobs {
		if matches, _ := filepath.Glob(filepath.Join(dir, glob)); len(matches) > 0 {
			return true
		}
	}
	return false
}

// zizmorFinding is the part of one entry of zizmor's report this stage reads.
type zizmorFinding struct {
	Ident          string `json:"ident"`
	Desc           string `json:"desc"`
	Determinations struct {
		Severity string `json:"severity"`
	} `json:"determinations"`
	Locations []zizmorLocation `json:"locations"`
}

// zizmorLocation is where a finding is. The nesting is zizmor's; path and line
// are the two facts worth reading out of it.
type zizmorLocation struct {
	Symbolic struct {
		Key struct {
			Local struct {
				Path string `json:"verbatim_path"`
			} `json:"Local"`
		} `json:"key"`
		Annotation string `json:"annotation"`
	} `json:"symbolic"`
	Concrete struct {
		Location struct {
			Start struct {
				Row int `json:"row"`
			} `json:"start_point"`
		} `json:"location"`
	} `json:"concrete"`
}

// path is the one the repository writes, without the leading "./" the auditor
// prefixes it with.
func (l zizmorLocation) path() string {
	return strings.TrimPrefix(l.Symbolic.Key.Local.Path, "./")
}

// line is the one an editor shows: zizmor counts rows from zero.
func (l zizmorLocation) line() int {
	return l.Concrete.Location.Start.Row + 1
}

// parseWorkflowFindings keeps the primary location of each finding. An
// unparseable report is an error: an auditor that did not run must never read
// as workflows that hold.
func parseWorkflowFindings(raw []byte) ([]WorkflowFinding, error) {
	var report []zizmorFinding

	if err := json.Unmarshal(raw, &report); err != nil {
		return nil, fmt.Errorf("read the audit report: %w", err)
	}
	if report == nil {
		return nil, fmt.Errorf("the auditor produced no report")
	}

	var findings []WorkflowFinding

	for _, entry := range report {
		if len(entry.Locations) == 0 {
			continue
		}
		at := entry.Locations[0]
		findings = append(findings, WorkflowFinding{
			Audit:    entry.Ident,
			Severity: strings.ToLower(entry.Determinations.Severity),
			File:     at.path(),
			Line:     at.line(),
			Detail:   cmp.Or(at.Symbolic.Annotation, entry.Desc),
		})
	}
	return findings, nil
}
