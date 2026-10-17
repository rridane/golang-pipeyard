package golang

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rridane/golang-pipeyard/x/mod"
)

// forge stands in for GitHub: the releases of one repository, and the content
// of their assets by id.
type forge struct {
	releases []mod.GithubPublishedRelease
	assets   map[int64]string
}

func (f forge) PublishedWithPrefix(_ context.Context, prefix string) ([]mod.GithubPublishedRelease, error) {
	var found []mod.GithubPublishedRelease
	for _, r := range f.releases {
		if strings.HasPrefix(r.Tag, prefix) {
			found = append(found, r)
		}
	}
	return found, nil
}

func (f forge) DownloadAsset(_ context.Context, id int64) ([]byte, error) {
	return []byte(f.assets[id]), nil
}

// Released out of order, beside another binary and a tag nobody can read: the
// last release is the highest version of this binary, not the last listed.
var released = forge{
	releases: []mod.GithubPublishedRelease{
		{Tag: "otlp-replay/v0.1.9", Assets: map[string]int64{"otlp-replay.sha256": 1}},
		{Tag: "otlp-replay/v0.1.10", Assets: map[string]int64{"otlp-replay.sha256": 2}},
		{Tag: "otlp-replay/vnext", Assets: map[string]int64{}},
		{Tag: "trace-transformer/v3.0.0", Assets: map[string]int64{"trace-transformer.sha256": 3}},
	},
	assets: map[int64]string{1: "old  otlp-replay\n", 2: "abc  otlp-replay\n", 3: "abc  trace-transformer\n"},
}

func TestDecide_SameDigestIsNotReleased(t *testing.T) {
	binary, err := decide(context.Background(), released, "otlp-replay", "abc")
	if err != nil {
		t.Fatal(err)
	}
	if binary != nil {
		t.Fatalf("an identical binary must not be released again, got %+v", binary)
	}
}

func TestDecide_ChangedBinaryRaisesThePatchOfTheHighestVersion(t *testing.T) {
	binary, err := decide(context.Background(), released, "otlp-replay", "def")
	if err != nil {
		t.Fatal(err)
	}
	want := &Binary{Name: "otlp-replay", Tag: "otlp-replay/v0.1.11", SHA256: "def"}
	if !reflect.DeepEqual(binary, want) {
		t.Fatalf("got %+v, want %+v", binary, want)
	}
}

func TestDecide_NeverReleasedStartsAt010(t *testing.T) {
	binary, err := decide(context.Background(), released, "pipeyard-builder", "abc")
	if err != nil {
		t.Fatal(err)
	}
	if binary == nil || binary.Tag != "pipeyard-builder/v0.1.0" {
		t.Fatalf("want pipeyard-builder/v0.1.0, got %+v", binary)
	}
}

// Without a checksum nothing proves the release holds this binary, so it is
// released again rather than assumed equal.
func TestDecide_ReleaseWithoutChecksumCountsAsChanged(t *testing.T) {
	bare := forge{releases: []mod.GithubPublishedRelease{{Tag: "otlp-replay/v1.2.3", Assets: map[string]int64{}}}}

	binary, err := decide(context.Background(), bare, "otlp-replay", "abc")
	if err != nil {
		t.Fatal(err)
	}
	if binary == nil || binary.Tag != "otlp-replay/v1.2.4" {
		t.Fatalf("want otlp-replay/v1.2.4, got %+v", binary)
	}
}

// What makes two builds of the same source byte for byte equal has to stay in
// the recipe, or every binary looks changed on every run.
func TestBinariesRecipe_IsReproducible(t *testing.T) {
	recipe := (&BinariesStage{}).recipe("cmd", []string{"a", "b"}).Render()

	for _, want := range []string{
		"FROM " + DefaultGoImage, "CGO_ENABLED=0", "GOFLAGS=-trimpath", "-buildvcs=false", "-buildid=",
		"-o /out/a ./cmd/a", "-o /out/b ./cmd/b",
	} {
		if !strings.Contains(recipe, want) {
			t.Errorf("the recipe lacks %q:\n%s", want, recipe)
		}
	}
}

func TestCommands_OneBinaryPerDirectory(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b", "a"} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	names, err := commands(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(names, []string{"a", "b"}) {
		t.Fatalf("got %v", names)
	}

	if _, err := commands(t.TempDir()); err == nil {
		t.Fatal("a directory with no command must be refused")
	}
}
