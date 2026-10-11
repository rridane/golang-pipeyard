package shell_executor

import (
	"context"
	"strings"
	"testing"
	"time"
)

// A failing command must fail the stage. This primitive used to return nil
// whatever happened, which made every shell step silently successful.
func TestRun_NonZeroExitIsAFailure(t *testing.T) {
	sh := ShellExecutor{}

	err := sh.Run(context.Background(), "exit 3")

	if err == nil {
		t.Fatal("a script exiting non-zero must fail")
	}
	if !strings.Contains(err.Error(), "status 3") {
		t.Fatalf("the exit status must be reported, got: %v", err)
	}
}

func TestRun_SuccessReturnsNil(t *testing.T) {
	sh := ShellExecutor{}

	if err := sh.Run(context.Background(), "true"); err != nil {
		t.Fatalf("a script exiting zero must succeed, got: %v", err)
	}
}

func TestRun_UnparsableScriptIsReported(t *testing.T) {
	sh := ShellExecutor{}

	if err := sh.Run(context.Background(), "if then fi ((("); err == nil {
		t.Fatal("a script that cannot be parsed must fail")
	}
}

func TestRun_TimeoutStopsTheScript(t *testing.T) {
	sh := ShellExecutor{Timeout: 100 * time.Millisecond}

	started := time.Now()
	err := sh.Run(context.Background(), "sleep 5")
	elapsed := time.Since(started)

	if err == nil {
		t.Fatal("a script exceeding Timeout must fail")
	}
	if elapsed > time.Second {
		t.Fatalf("Timeout must stop the script, it ran for %s", elapsed)
	}
}

// Output is what lets a stage read a value computed by a command.
func TestOutput_CapturesStdout(t *testing.T) {
	sh := ShellExecutor{}

	out, err := sh.Output(context.Background(), "echo v1.4.0")
	if err != nil {
		t.Fatal(err)
	}

	if strings.TrimSpace(string(out)) != "v1.4.0" {
		t.Fatalf("want %q, got %q", "v1.4.0", string(out))
	}
}

func TestOutput_ReportsFailureAndKeepsWhatWasWritten(t *testing.T) {
	sh := ShellExecutor{}

	out, err := sh.Output(context.Background(), "echo partial; exit 1")

	if err == nil {
		t.Fatal("a failing script must fail even when it wrote output")
	}
	if !strings.Contains(string(out), "partial") {
		t.Fatalf("output written before the failure must be kept, got %q", string(out))
	}
}
