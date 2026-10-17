package common

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
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

	// Accept are the advisories this repository has assessed and holds not
	// to apply: a vulnerability in a package nothing imports, or one with no
	// fixed version. Each entry belongs next to the reason it was accepted,
	// which is why they are declared here and not on a command line.
	//
	// An identifier that matches nothing is an error. An exemption nobody
	// needs any more is one nobody rereads.
	Accept []string

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

	// Aliases are the other identifiers the databases give it.
	Aliases []string
}

// names is every identifier this vulnerability answers to.
func (a Advisory) names() []string {
	return append([]string{a.ID}, a.Aliases...)
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

	accepted, remaining := s.sift(advisories)

	for _, advisory := range remaining {
		log.Error("advisory", zap.String("dependency", advisory.String()))
	}
	for _, advisory := range accepted {
		log.Info("advisory accepted", zap.String("dependency", advisory.String()))
	}

	unused := s.unusedAccepts(accepted)
	for _, id := range unused {
		log.Error("accepted advisory no longer applies", zap.String("advisory", id))
	}

	switch {
	case len(remaining) == 0 && len(unused) == 0:
		log.Info("no known vulnerability in the locked dependencies",
			zap.Strings("paths", paths), zap.Int("accepted", len(accepted)))
		return nil
	case s.WarnOnly:
		log.Warn("advisories found, reported only", zap.Int("advisories", len(remaining)))
		return nil
	case len(unused) > 0:
		return fmt.Errorf("%d accepted advisory(ies) no longer apply, remove them: %s",
			len(unused), strings.Join(unused, " "))
	default:
		return fmt.Errorf("%d advisory(ies) against the locked dependencies", len(remaining))
	}
}

// sift splits the advisories this repository has accepted from the rest.
func (s *AuditStage) sift(advisories []Advisory) (accepted, remaining []Advisory) {
	for _, advisory := range advisories {
		if s.accepts(advisory) {
			accepted = append(accepted, advisory)
			continue
		}
		remaining = append(remaining, advisory)
	}
	return accepted, remaining
}

// accepts matches an accepted identifier against the vulnerability's own id
// and its aliases alike, so an exemption holds whichever one the databases
// lead with.
func (s *AuditStage) accepts(advisory Advisory) bool {
	for _, name := range advisory.names() {
		if slices.Contains(s.Accept, name) {
			return true
		}
	}
	return false
}

// unusedAccepts are the accepted identifiers that matched nothing: either the
// advisory is gone, or it never applied to this repository.
func (s *AuditStage) unusedAccepts(accepted []Advisory) []string {
	matched := make(map[string]bool)
	for _, advisory := range accepted {
		for _, name := range advisory.names() {
			matched[name] = true
		}
	}

	var unused []string
	for _, id := range s.Accept {
		if !matched[id] {
			unused = append(unused, id)
		}
	}
	return unused
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
					Aliases:  vulnerability.Aliases,
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
