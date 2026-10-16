package python

import (
	"slices"
	"strings"
	"testing"
)

// uv's cache is always kept, and a stage's own caches come after it.
func TestToolchain_AlwaysKeepsTheUvCache(t *testing.T) {
	tool := Toolchain{Caches: []string{"/root/.cache/mine"}}.Tool("x", "ruff check")

	if !slices.Equal(tool.Caches, []string{uvCache, "/root/.cache/mine"}) {
		t.Fatalf("got %v", tool.Caches)
	}
}

// The Debian packages and uv go in one RUN, so the image carries one layer
// rather than three.
func TestToolchain_InstallsItsPackagesInOneStep(t *testing.T) {
	setup := Toolchain{Packages: []string{"libatomic1"}}.Tool("x", "pyright").Setup

	if strings.Count(setup, "RUN ") != 1 {
		t.Fatalf("want a single RUN, got %q", setup)
	}
	for _, want := range []string{"libatomic1", "rm -rf /var/lib/apt/lists/*", "uv==" + DefaultUvVersion} {
		if !strings.Contains(setup, want) {
			t.Errorf("setup should contain %q:\n%s", want, setup)
		}
	}
}

// Without a package to install there is no reason to touch apt at all.
func TestToolchain_SkipsAptWhenNothingNeedsIt(t *testing.T) {
	if setup := (Toolchain{}).Tool("x", "ruff").Setup; strings.Contains(setup, "apt-get") {
		t.Fatalf("apt should not be invoked: %q", setup)
	}
}

// An unresolved import reads as an error against the code, so a stage that
// examines the code has to install what the code imports.
// The zero value installs them, and a caller who opts out is obeyed — the
// three stages used to overwrite this field, which made it a lie.
func TestToolchain_ExtrasAreInstalledUnlessRefused(t *testing.T) {
	byDefault := Toolchain{}.Tool("x", "pyright").Script
	refused := Toolchain{WithoutExtras: true}.Tool("x", "pyright").Script

	if !strings.Contains(byDefault, "uv run --frozen --all-extras pyright") {
		t.Errorf("got %q", byDefault)
	}
	if strings.Contains(refused, "--all-extras") {
		t.Errorf("got %q", refused)
	}
}

func TestPytestStage_Command(t *testing.T) {
	stage := &PytestStage{Paths: []string{"tests"}}

	if got, want := stage.command("not ray"), "pytest -m 'not ray' 'tests'"; got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
	if got, want := (&PytestStage{}).command(""), "pytest"; got != want {
		t.Errorf("bare: got %q, want %q", got, want)
	}
}

// Two runs have to produce the same script, or BuildKit rebuilds for nothing.
func TestPytestStage_ExportsAreSortedAndQuoted(t *testing.T) {
	stage := &PytestStage{Env: map[string]string{"RAY_ADDRESS": "local", "A_FLAG": "with space"}}

	if got, want := stage.exports(), "export A_FLAG='with space'\nexport RAY_ADDRESS='local'\n"; got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

// The field is the repository's choice of markers; the flag lets one run
// override it.
func TestPytestStage_MarkersFlagDefaultsToTheField(t *testing.T) {
	flag, found := (&PytestStage{Markers: "not ray"}).Configure().Flag("markers")
	if !found {
		t.Fatal("no --markers")
	}
	if flag.Default != "not ray" {
		t.Errorf("got %v", flag.Default)
	}
}

func TestArchitectureStage_Command(t *testing.T) {
	if got, want := (&ArchitectureStage{}).command(), "lint-imports"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	stage := &ArchitectureStage{Config: "setup.cfg"}
	if got, want := stage.command(), "lint-imports --config 'setup.cfg'"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
