package common

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"github.com/rridane/golang-pipeyard/x/stack"
	"go.uber.org/zap"
)

// defaultIgnorePath is where the scanner looks for the accepted findings.
const defaultIgnorePath = ".gitleaksignore"

// SecretsStage looks for secrets in the whole history, not only the checkout:
// a credential removed in a later commit is still readable by anyone who
// clones.
type SecretsStage struct {
	// OutDir is where the stage may write. Empty uses stack.DefaultOutDir.
	OutDir string

	// IgnorePath lists the accepted findings. It is read from the checkout, so
	// an exception holds before it is committed. Empty uses ".gitleaksignore".
	IgnorePath string

	// Image is the scanner. Empty uses DefaultGitleaksImage.
	Image string
}

// Finding locates one secret. The scanner runs redacted, so the secret itself
// never reaches the report, the logs or the terminal.
type Finding struct {
	RuleID    string `json:"RuleID"`
	File      string `json:"File"`
	StartLine int    `json:"StartLine"`
	Commit    string `json:"Commit"`
}

// ShortCommit is the commit as a human writes it.
func (f Finding) ShortCommit() string {
	if len(f.Commit) < 8 {
		return f.Commit
	}
	return f.Commit[:8]
}

func (s *SecretsStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("secrets")
	spec.SetDescription("Look for secrets in the whole history")
	return spec
}

func (s *SecretsStage) Execute(
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

	// A bundle carries the whole history in one file, and git produces it —
	// which is what makes this work from a worktree, where .git is a file
	// pointing elsewhere.
	host, bundle, err := workspace.Generated("history.bundle")
	if err != nil {
		return err
	}
	if err := sh.Run(ctx, "git bundle create "+stack.Quote(host)+" --all"); err != nil {
		return fmt.Errorf("bundle the history: %w", err)
	}

	result, err := stack.Tool{
		Name:   "secrets",
		Image:  cmp.Or(s.Image, DefaultGitleaksImage),
		Script: s.script(bundle),
	}.Run(ctx, cb, workspace)
	if err != nil {
		return err
	}

	var findings []Finding
	if err := json.Unmarshal([]byte(result.Output), &findings); err != nil {
		return fmt.Errorf("the scanner's output is not JSON (exit %d):\n%s", result.ExitCode, result.Output)
	}

	for _, f := range findings {
		log.Error("secret in the history",
			zap.String("rule", f.RuleID),
			zap.String("file", f.File),
			zap.Int("line", f.StartLine),
			zap.String("commit", f.ShortCommit()))
	}
	if len(findings) > 0 {
		return fmt.Errorf("%d secret(s) found in the history", len(findings))
	}

	log.Info("no secret in the history")
	return nil
}

// script clones the bundle and scans it. Only the report reaches stdout so the
// stage can parse it, and the exit code is forced to zero: a finding is not a
// failure of the scanner, and this stage decides what it means.
func (s *SecretsStage) script(bundle string) string {
	return fmt.Sprintf(
		"git -c init.defaultBranch=main clone -q %s /history >/dev/null 2>&1 && "+
			"gitleaks git /history --log-opts=--all --redact --no-banner --exit-code 0 "+
			"--gitleaks-ignore-path %s "+
			"--report-format json --report-path /tmp/report.json >/dev/null 2>&1 && "+
			"cat /tmp/report.json\n",
		stack.Quote(bundle), stack.Quote(cmp.Or(s.IgnorePath, defaultIgnorePath)))
}
