package common

import (
	"cmp"
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
// answers for what a build would install. Which lockfiles is the scanner's
// business — uv.lock, go.sum and the rest are found in the same pass.
type AuditStage struct {
	// OutDir is where the stage may write. Empty uses stack.DefaultOutDir.
	OutDir string

	// Paths are the directories to scan, relative to the repository root.
	// Empty scans the whole repository.
	Paths []string

	// Image is the scanner, pinned. Empty uses DefaultScannerImage.
	Image string

	// WarnOnly reports the advisories and succeeds. An advisory is not always
	// actionable the day it is published.
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
	workspace, err := stack.NewWorkspace(ctx, sh, cmp.Or(s.OutDir, stack.DefaultOutDir))
	if err != nil {
		return err
	}

	paths := s.Paths
	if len(paths) == 0 {
		paths = []string{"."}
	}

	result, err := stack.Tool{
		Name:   "audit",
		Image:  cmp.Or(s.Image, DefaultScannerImage),
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

// script writes the report to a file and prints it, because the scanner
// narrates its walk on stderr and the output has to be JSON and nothing else.
// The exit code is left aside: the report says how many advisories there are,
// and a report that will not parse says the scanner never ran.
func (s *AuditStage) script(paths []string) string {
	return fmt.Sprintf(
		"%s scan source --recursive %s --format json --output-file /tmp/report.json 2>/dev/null\n"+
			"cat /tmp/report.json\n",
		scannerBinary, stack.QuoteAll(paths))
}

// parseAdvisories flattens the scanner's report, which nests a lockfile, the
// packages it locks and the vulnerabilities against each.
//
// One vulnerability comes back several times, once under each identifier the
// databases give it: a GO- entry, its CVE- and its GHSA- alias are the same
// hole. Counting entries would multiply every number by three, so the aliases
// are what decides whether two entries are one advisory.
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
					ID      string   `json:"id"`
					Aliases []string `json:"aliases"`
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
			// Per package: the same identifier against two packages is two
			// advisories, because each is fixed on its own.
			seen := make(map[string]bool)

			for _, vulnerability := range locked.Vulnerabilities {
				names := append([]string{vulnerability.ID}, vulnerability.Aliases...)
				if known(seen, names) {
					continue
				}
				for _, name := range names {
					seen[name] = true
				}

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

// known reports whether any of the names has already been counted.
func known(seen map[string]bool, names []string) bool {
	for _, name := range names {
		if seen[name] {
			return true
		}
	}
	return false
}
