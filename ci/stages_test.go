package main

import (
	"testing"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/stack/golang"
)

// Every stage this pipeline registers. A stage missing from here is a stage
// nothing checks.
func stages() []pi.Stage {
	return []pi.Stage{
		&golang.BuildStage{}, &golang.FormatStage{}, &golang.VetStage{},
		&golang.TestStage{}, &golang.LintStage{}, &ReleaseStage{}, &PublishStage{},
	}
}

// The command name is what a workflow types. Renaming one breaks every caller
// silently, so it is worth a test of its own.
func TestStages_AreNamedAsTheWorkflowCallsThem(t *testing.T) {
	want := map[string]bool{
		"build": true, "fmt": true, "vet": true,
		"test": true, "lint": true, "release": true, "publish": true,
	}

	for _, stage := range stages() {
		name := stage.Configure().Command()
		if !want[name] {
			t.Errorf("unexpected stage %q", name)
		}
		delete(want, name)
	}

	for name := range want {
		t.Errorf("stage %q is registered nowhere", name)
	}
}

func TestStages_AllDescribeThemselves(t *testing.T) {
	for _, stage := range stages() {
		spec := stage.Configure()
		if spec.Short() == "" {
			t.Errorf("stage %q has no description; it shows in --help", spec.Command())
		}
	}
}

// The flags are the stage's contract with the workflows that call it.
func TestStages_Flags(t *testing.T) {
	for _, tc := range []struct {
		stage    pi.Stage
		flag     string
		kind     ru.FlagType
		fallback any
		required bool
	}{
		{&golang.TestStage{}, "pkg", ru.StringFlag, "./...", false},
		{&golang.TestStage{}, "race", ru.BoolFlag, true, false},
		{&golang.TestStage{}, "timeout", ru.DurationFlag, "5m", false},
		{&golang.BuildStage{}, "pkg", ru.StringFlag, "./...", false},
		{&golang.VetStage{}, "pkg", ru.StringFlag, "./...", false},
		{&ReleaseStage{}, "version", ru.StringFlag, "", true},
		{&ReleaseStage{}, "image", ru.StringFlag, "ghcr.io/rridane/pipeyard", false},
		{&ReleaseStage{}, "push", ru.BoolFlag, false, false},
		{&PublishStage{}, "version", ru.StringFlag, "", true},
		{&PublishStage{}, "asset", ru.StringFlag, "dist/pipeyard-ci", false},
	} {
		stage := tc.stage.Configure()

		flag, found := stage.Flag(tc.flag)
		if !found {
			t.Errorf("%s: no flag %q", stage.Command(), tc.flag)
			continue
		}
		if flag.Type != tc.kind {
			t.Errorf("%s --%s: type %q, want %q", stage.Command(), tc.flag, flag.Type, tc.kind)
		}
		if flag.Default != tc.fallback {
			t.Errorf("%s --%s: default %v, want %v", stage.Command(), tc.flag, flag.Default, tc.fallback)
		}
		if flag.Required != tc.required {
			t.Errorf("%s --%s: required %v, want %v", stage.Command(), tc.flag, flag.Required, tc.required)
		}
	}
}

// A version written either way has to produce the tag Git carries.
func TestReleaseTag(t *testing.T) {
	for version, want := range map[string]string{
		"0.1.0":  "v0.1.0",
		"v0.1.0": "v0.1.0",
		"1.10.3": "v1.10.3",
	} {
		if got := releaseTag(version); got != want {
			t.Errorf("releaseTag(%q) = %q, want %q", version, got, want)
		}
	}
}

func TestRegistryFromEnv(t *testing.T) {
	t.Setenv("PIPEYARD_REGISTRY_USER", "someone")
	t.Setenv("PIPEYARD_REGISTRY_TOKEN", "secret")
	t.Setenv("PIPEYARD_REGISTRY_HOST", "ghcr.io")

	credential, ok := registryFromEnv()["ghcr.io"]
	if !ok {
		t.Fatal("want a credential under ghcr.io")
	}
	if credential.Username != "someone" || credential.Password != "secret" {
		t.Fatalf("unexpected credential %+v", credential)
	}

	// Half a credential is no credential: the builder falls back to the
	// machine's docker config rather than failing to authenticate later.
	t.Setenv("PIPEYARD_REGISTRY_TOKEN", "")

	if got := registryFromEnv(); got != nil {
		t.Fatalf("a half-set credential must be ignored, got %v", got)
	}
}
