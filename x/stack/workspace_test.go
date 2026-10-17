package stack

import "testing"

func TestLeftOut(t *testing.T) {
	for name, want := range map[string]bool{
		".dockerignore":               true,
		"out/workspace/go.mod":        true,
		".github/workflows/ci.yml":    false,
		"docker/.dockerignore":        false,
		"outline.md":                  false,
		"src/out/generated_module.py": false,
	} {
		if got := leftOut(name, "out"); got != want {
			t.Errorf("leftOut(%q) = %v, want %v", name, got, want)
		}
	}
}
