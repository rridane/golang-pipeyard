package main

import (
	"strings"
	"testing"
)

// The recipe is what ships. A template that renders into something BuildKit
// cannot parse would only fail on a tag, which is the worst place to find out.
func TestReleaseRecipe_Parses(t *testing.T) {
	r := release{version: "1.2.3", image: "example/pipeyard"}

	stages, err := r.recipe().Stages()
	if err != nil {
		t.Fatalf("the release recipe must parse: %v", err)
	}

	if len(stages) != 2 {
		t.Fatalf("want a builder and a runtime stage, got %d", len(stages))
	}

	if stages[1].Name != "runtime" {
		t.Fatalf("the exported stage must be named runtime, got %q", stages[1].Name)
	}
}

// The runtime user has to own the caches the Go toolchain writes, or every
// stage that shells out to go fails inside the published image.
func TestReleaseRecipe_RuntimeUserOwnsItsCaches(t *testing.T) {
	rendered := release{version: "1.2.3"}.recipe().Render()

	runtime := rendered[strings.Index(rendered, "AS runtime"):]

	for _, want := range []string{"-h /home/ci", "chown -R ci /home/ci /go", "USER ci"} {
		if !strings.Contains(runtime, want) {
			t.Errorf("the runtime stage must contain %q", want)
		}
	}
}
