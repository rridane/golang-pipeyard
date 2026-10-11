package lay

import (
	"strings"
	"testing"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
)

// The commit title is the one piece too long to repeat on every line, so the
// opening line carries it.
func TestOpening_CarriesTheCommitTitle(t *testing.T) {
	t.Setenv(pi.CommitTitleEnvVar, "send pipeline telemetry from the CI")

	want := `started "send pipeline telemetry from the CI"`
	if got := (&StageObserver{}).opening(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// Run from a laptop with nothing set, the line must not carry empty decoration.
func TestOpening_WithoutAnIdentity(t *testing.T) {
	t.Setenv(pi.CommitTitleEnvVar, "")

	if got, want := (&StageObserver{}).opening(), "started"; got != want {
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
