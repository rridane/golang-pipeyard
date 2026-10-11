package otel

import (
	"testing"

	"github.com/rridane/golang-pipeyard/pi"
	"go.uber.org/zap/zapcore"
)

// A log viewer shows the body and collapses the attributes, and lines from
// several runs interleave there, so every body has to locate itself.
func TestBody_CarriesTheStageAndTheRun(t *testing.T) {
	t.Setenv(pi.BranchEnvVar, "main")
	t.Setenv(pi.CommitSHAEnvVar, "f9d05bdbd2186129927844c16bff54bdb63382ed")

	core := &otelZapCore{run: runScope()}
	entry := zapcore.Entry{Message: "every package is formatted"}

	// The stage from the logger the injector handed the stage.
	core.fields = []zapcore.Field{{Key: "stage", Type: zapcore.StringType, String: "fmt"}}

	if got, want := core.body(entry, nil), "[fmt main@f9d05bd] every package is formatted"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// The stage from the call, which is how the observer writes its own lines.
	core.fields = nil
	call := []zapcore.Field{{Key: "stage", Type: zapcore.StringType, String: "vet"}}

	if got, want := core.body(entry, call), "[vet main@f9d05bd] every package is formatted"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Run from a laptop with nothing set, a body must carry no empty brackets.
func TestBody_WithoutAnIdentity(t *testing.T) {
	t.Setenv(pi.BranchEnvVar, "")
	t.Setenv(pi.CommitSHAEnvVar, "")

	core := &otelZapCore{run: runScope()}
	entry := zapcore.Entry{Message: "every package is formatted"}

	if got, want := core.body(entry, nil), "every package is formatted"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRunScope(t *testing.T) {
	for _, tc := range []struct{ branch, sha, want string }{
		{"main", "f9d05bdbd2186129927844c16bff54bdb63382ed", "main@f9d05bd"},
		{"main", "", "main"},
		{"", "f9d05bdbd2186129927844c16bff54bdb63382ed", "@f9d05bd"},
		{"", "", ""},
	} {
		t.Setenv(pi.BranchEnvVar, tc.branch)
		t.Setenv(pi.CommitSHAEnvVar, tc.sha)

		if got := runScope(); got != tc.want {
			t.Errorf("branch=%q sha=%q: got %q, want %q", tc.branch, tc.sha, got, tc.want)
		}
	}
}
