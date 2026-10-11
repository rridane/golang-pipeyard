package lay

import (
	"strings"
	"testing"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
)

// One line per run has to be readable on its own, with its fields collapsed.
func TestOpening_NamesTheStageAndTheChange(t *testing.T) {
	t.Setenv(pi.BranchEnvVar, "main")
	t.Setenv(pi.CommitSHAEnvVar, "e8d9f62a41caa0a259da521a0d84cf5ab85faf97")
	t.Setenv(pi.CommitTitleEnvVar, "send pipeline telemetry from the CI")

	observer := &StageObserver{}

	want := `vet started on main @e8d9f62 "send pipeline telemetry from the CI"`
	if got := observer.opening("vet"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// Run from a laptop with nothing set, the line must not carry empty decoration.
func TestOpening_WithoutAnIdentity(t *testing.T) {
	t.Setenv(pi.BranchEnvVar, "")
	t.Setenv(pi.CommitSHAEnvVar, "")
	t.Setenv(pi.CommitTitleEnvVar, "")

	if got, want := (&StageObserver{}).opening("fmt"), "fmt started"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// A Prometheus store keeps resource attributes out of the series, so a branch
// that is only on the resource cannot be filtered on.
func TestSeriesAttributes_CarryTheBranch(t *testing.T) {
	t.Setenv(pi.BranchEnvVar, "main")

	attrs := (&StageObserver{}).seriesAttributes("deploy", ru.ParsedArgs{})

	var keys []string
	for _, attr := range attrs {
		keys = append(keys, string(attr.Key))
	}

	if !strings.Contains(strings.Join(keys, ","), "branch") {
		t.Fatalf("want branch among the series attributes, got %v", keys)
	}
}

// The commit never reaches the metrics: as a label it is one series per commit.
func TestSeriesAttributes_CarryNoCommit(t *testing.T) {
	t.Setenv(pi.CommitSHAEnvVar, "e8d9f62a41caa0a259da521a0d84cf5ab85faf97")
	t.Setenv(pi.CommitTitleEnvVar, "a title")

	for _, attr := range (&StageObserver{}).seriesAttributes("deploy", ru.ParsedArgs{}) {
		if strings.HasPrefix(string(attr.Key), "commit") {
			t.Errorf("the metrics must carry no commit, got %s", attr.Key)
		}
	}
}
