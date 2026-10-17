package stack

import "testing"

func TestParseReport_SplitsTheOutputFromTheExitCode(t *testing.T) {
	result, err := parseReport([]byte("first line\nsecond line\nexit=2\n"))
	if err != nil {
		t.Fatal(err)
	}

	if result.Output != "first line\nsecond line" {
		t.Fatalf("the output should stop before the exit code, got %q", result.Output)
	}
	if result.ExitCode != 2 {
		t.Fatalf("want exit 2, got %d", result.ExitCode)
	}
	if !result.Failed() {
		t.Fatal("a non-zero exit is a failure")
	}
}

// A tool that printed nothing still reports how it exited, and the stage must
// not read an empty output as a missing report.
func TestParseReport_AcceptsNoOutputAtAll(t *testing.T) {
	result, err := parseReport([]byte("exit=0\n"))
	if err != nil {
		t.Fatal(err)
	}

	if result.Output != "" {
		t.Fatalf("want no output, got %q", result.Output)
	}
	if result.Failed() {
		t.Fatal("exit 0 is a success")
	}
}

// zizmor prints its JSON without a final newline, so the trailer lands on the
// same line as the output's last character.
func TestParseReport_TrailerAfterOutputWithoutNewline(t *testing.T) {
	result, err := parseReport([]byte("[]exit=13\n"))
	if err != nil {
		t.Fatal(err)
	}

	if result.Output != "[]" || result.ExitCode != 13 {
		t.Fatalf("want [] and exit 13, got %q and %d", result.Output, result.ExitCode)
	}
}

// Without the trailer the stage has no idea whether the tool ran, so this has
// to be an error rather than a zero value.
func TestParseReport_RefusesAReportWithoutATrailer(t *testing.T) {
	if _, err := parseReport([]byte("the tool crashed\n")); err == nil {
		t.Fatal("a report without an exit code must be refused")
	}
}

// A cache mounted on a path the image does not read is never used, and
// nothing reports it: the mount succeeds and the tool starts cold every time.
// So the rendering is worth pinning, even though what it renders is one line.
func TestToolMounts(t *testing.T) {
	if got := (Tool{}).mounts(); got != "" {
		t.Fatalf("no cache should render nothing, got %q", got)
	}

	got := Tool{Caches: []string{"/go/pkg/mod", "/root/.cache/go-build"}}.mounts()
	want := " --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build"

	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}
