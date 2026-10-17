package common

import (
	"strings"
	"testing"
)

// What zizmor printed for two checkouts that keep their token, cut to the
// fields read. The second carries the "ignore" comment the workflow used to
// silence it, which --no-ignores reports and marks rather than dropping.
const workflowReport = `[
  {"ident":"artipacked","desc":"credential persistence through GitHub Actions artifacts",
   "determinations":{"confidence":"Low","severity":"Medium","persona":"Regular"},
   "locations":[{"symbolic":{"key":{"Local":{"verbatim_path":"./.github/workflows/ci.yml"}},
                             "annotation":"does not set persist-credentials: false"},
                 "concrete":{"location":{"start_point":{"row":29,"column":8}}}}],
   "ignored":false},
  {"ident":"template-injection","desc":"code injection via template expansion",
   "determinations":{"severity":"High"},
   "locations":[{"symbolic":{"key":{"Local":{"verbatim_path":"./.github/workflows/ci.yml"}}},
                 "concrete":{"location":{"start_point":{"row":40,"column":8}}}}],
   "ignored":true}
]`

func TestParseWorkflowFindings(t *testing.T) {
	findings, err := parseWorkflowFindings([]byte(workflowReport))
	if err != nil {
		t.Fatal(err)
	}

	// The line is the one an editor shows, the path the one the repository
	// writes, and the detail falls back to the audit's own description when
	// the location carries no annotation.
	want := []string{
		".github/workflows/ci.yml:30 artipacked (medium): does not set persist-credentials: false",
		".github/workflows/ci.yml:41 template-injection (high): code injection via template expansion",
	}
	if len(findings) != len(want) {
		t.Fatalf("got %d findings, want %d: %v", len(findings), len(want), findings)
	}
	for i, w := range want {
		if got := findings[i].String(); got != w {
			t.Errorf("got  %q\nwant %q", got, w)
		}
	}
}

func TestParseWorkflowFindings_EmptyReportIsClean(t *testing.T) {
	findings, err := parseWorkflowFindings([]byte(`[]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("want none, got %v", findings)
	}
}

// The auditor refusing to run must not read as workflows that hold.
func TestParseWorkflowFindings_RefusesWhatIsNotAReport(t *testing.T) {
	for _, output := range []string{"", "null", "error: no inputs collected"} {
		if _, err := parseWorkflowFindings([]byte(output)); err == nil {
			t.Errorf("%q must not read as clean workflows", output)
		}
	}
}

// A report whose shape is not the one this stage reads has to say so. The day
// the auditor changes its JSON, the message is the only clue there will be.
func TestParseWorkflowFindings_SaysWhyAReportIsUnreadable(t *testing.T) {
	_, err := parseWorkflowFindings([]byte(`[{"locations":"not a list"}]`))
	if err == nil {
		t.Fatal("a report of the wrong shape must be an error")
	}
	if !strings.Contains(err.Error(), "locations") {
		t.Errorf("the message should name what it choked on, got %q", err)
	}
}

// A policy replaces the repository's own configuration, and no comment left in
// a workflow may exempt it from one of its audits.
func TestWorkflowsScript(t *testing.T) {
	stage := &WorkflowsStage{}

	if got := stage.script(""); strings.Contains(got, "--config") {
		t.Errorf("no policy, no --config: %q", got)
	}
	if got := stage.script("/src/.pipeyard/zizmor.yml"); !strings.Contains(got, "--config '/src/.pipeyard/zizmor.yml' ") {
		t.Errorf("the policy should be passed: %q", got)
	}
	if got := stage.script(""); !strings.Contains(got, "--no-ignores") {
		t.Errorf("an ignore comment must not exempt a workflow: %q", got)
	}
}
