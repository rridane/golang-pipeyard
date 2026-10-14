package common

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"github.com/rridane/golang-pipeyard/x/stack"
	"go.uber.org/zap"
)

// DefaultScannerImage carries the vulnerability scanner, pinned so a laptop
// and the CI read the same database client. The image puts its binary at
// /osv-scanner, outside PATH.
const DefaultScannerImage = "ghcr.io/google/osv-scanner:v2.6.0"

const scannerBinary = "/osv-scanner"

// AuditStage looks the repository's locked dependencies up in the OSV
// database. It reads the lockfiles rather than the installed packages, so it
// answers for what a build would install and not for what this machine
// happens to have.
//
// Which lockfiles is the scanner's business: uv.lock, go.sum,
// package-lock.json and the rest are found in the same pass, so the stage is
// language-agnostic without knowing it.
type AuditStage struct {
	// OutDir is where the stage may write. Empty uses stack.DefaultOutDir.
	OutDir string

	// Paths are the directories to scan, relative to the repository root.
	// Empty scans the whole repository.
	Paths []string

	// Image is the scanner, pinned. Empty uses DefaultScannerImage.
	Image string

	// WarnOnly reports the advisories and succeeds. An advisory is not always
	// actionable the day it is published, and a repository that fails on
	// every one of them is a repository whose pipeline gets bypassed.
	WarnOnly bool
}

// Advisory is one vulnerability against one locked package.
type Advisory struct {
	Lockfile string
	Package  string
	Version  string
	ID       string
}

func (a Advisory) String() string {
	return fmt.Sprintf("%s %s: %s (%s)", a.Package, a.Version, a.ID, a.Lockfile)
}

func (s *AuditStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("audit")
	spec.SetDescription("Look the locked dependencies up in the OSV database")
	return spec
}

func (s *AuditStage) Execute(
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
		Name:   "audit",
		Image:  s.image(),
		Script: s.script(paths),
	}.Run(ctx, cb, workspace)
	if err != nil {
		return err
	}

	advisories, err := parseAdvisories([]byte(result.Output))
	if err != nil {
		return fmt.Errorf("%w (exit %d):\n%s", err, result.ExitCode, result.Output)
	}

	for _, a := range advisories {
		log.Error("advisory", zap.String("dependency", a.String()))
	}

	switch {
	case len(advisories) == 0:
		log.Info("no known vulnerability in the locked dependencies",
			zap.Strings("paths", paths))
		return nil
	case s.WarnOnly:
		log.Warn("advisories found, reported only", zap.Int("advisories", len(advisories)))
		return nil
	default:
		return fmt.Errorf("%d advisory(ies) against the locked dependencies", len(advisories))
	}
}

// script writes the report to a file and prints that file, because the
// scanner narrates its walk on stderr and the stage's output has to be JSON
// and nothing else.
//
// The scanner's exit code is left aside: the report is what says how many
// advisories there are, and a report that will not parse is what says the
// scanner never ran.
func (s *AuditStage) script(paths []string) string {
	quoted := make([]string, 0, len(paths))
	for _, p := range paths {
		quoted = append(quoted, stack.Quote(p))
	}

	return fmt.Sprintf(
		"%s scan source --recursive %s --format json --output-file /tmp/report.json 2>/dev/null\n"+
			"cat /tmp/report.json\n",
		scannerBinary, strings.Join(quoted, " "))
}

// parseAdvisories flattens the scanner's report, which nests a lockfile, the
// packages it locks and the vulnerabilities against each.
func parseAdvisories(raw []byte) ([]Advisory, error) {
	var report struct {
		Results []struct {
			Source struct {
				Path string `json:"path"`
			} `json:"source"`
			Packages []struct {
				Package struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				} `json:"package"`
				Vulnerabilities []struct {
					ID string `json:"id"`
				} `json:"vulnerabilities"`
			} `json:"packages"`
		} `json:"results"`
	}

	if err := json.Unmarshal(raw, &report); err != nil {
		return nil, fmt.Errorf("the scanner produced no report")
	}

	var advisories []Advisory
	for _, result := range report.Results {
		lockfile := strings.TrimPrefix(result.Source.Path, stack.ContainerRoot+"/")

		for _, locked := range result.Packages {
			for _, vulnerability := range locked.Vulnerabilities {
				advisories = append(advisories, Advisory{
					Lockfile: lockfile,
					Package:  locked.Package.Name,
					Version:  locked.Package.Version,
					ID:       vulnerability.ID,
				})
			}
		}
	}
	return advisories, nil
}

func (s *AuditStage) paths() []string {
	if len(s.Paths) == 0 {
		return []string{"."}
	}
	return s.Paths
}

func (s *AuditStage) outDir() string {
	if s.OutDir == "" {
		return stack.DefaultOutDir
	}
	return s.OutDir
}

func (s *AuditStage) image() string {
	if s.Image == "" {
		return DefaultScannerImage
	}
	return s.Image
}
