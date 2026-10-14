package common

import (
	"strings"
	"testing"
	"testing/fstest"
)

// tree is a repository as the guard sees it: a few files, one of them a
// model's weights, one of them holding a word nobody should have written.
func tree() fstest.MapFS {
	return fstest.MapFS{
		"src/model.py":         {Data: []byte("import torch\n")},
		"models/latest.ckpt":   {Data: []byte("weights")},
		"data/raw/spans.pq":    {Data: []byte("columns")},
		"docs/notes.md":        {Data: []byte("the host is prod-db-01\nnothing else\n")},
		"src/big.py":           {Data: []byte(strings.Repeat("x", 4096))},
		"deleted/from/tree.py": {Data: []byte("")},
	}
}

func files() []string {
	return []string{
		"src/model.py",
		"models/latest.ckpt",
		"data/raw/spans.pq",
		"docs/notes.md",
		"src/big.py",
	}
}

func TestInspect_RefusesAForbiddenName(t *testing.T) {
	guard := GuardStage{ForbiddenPaths: []string{"*.ckpt"}}

	violations, err := guard.Inspect(tree(), files(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 {
		t.Fatalf("want one violation, got %v", violations)
	}
	if violations[0].Path != "models/latest.ckpt" {
		t.Fatalf("the weights should be what was refused, got %q", violations[0].Path)
	}
}

// A pattern naming a directory must catch what is inside it, which is the
// form a data directory takes.
func TestInspect_RefusesAForbiddenDirectory(t *testing.T) {
	guard := GuardStage{ForbiddenPaths: []string{"data/raw/*"}}

	violations, err := guard.Inspect(tree(), files(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 || violations[0].Path != "data/raw/spans.pq" {
		t.Fatalf("want the parquet refused, got %v", violations)
	}
}

func TestInspect_RefusesAFileOverTheCeiling(t *testing.T) {
	guard := GuardStage{MaxFileSize: 1024}

	violations, err := guard.Inspect(tree(), files(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range violations {
		if v.Path == "src/big.py" {
			return
		}
	}
	t.Fatalf("the oversized file should have been refused, got %v", violations)
}

// The message must locate the word without repeating it, because the stage's
// output is read by whoever happens to open the job.
func TestInspect_ReportsAForbiddenWordWithoutWritingIt(t *testing.T) {
	guard := GuardStage{}

	violations, err := guard.Inspect(tree(), files(), []string{"prod-db-01"})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 {
		t.Fatalf("want one violation, got %v", violations)
	}

	v := violations[0]
	if v.Path != "docs/notes.md" {
		t.Fatalf("want the note, got %q", v.Path)
	}
	if !strings.Contains(v.Reason, "line 1") {
		t.Fatalf("the reason should give the line, got %q", v.Reason)
	}
	if strings.Contains(v.Reason, "prod-db-01") {
		t.Fatalf("the reason leaks the word itself: %q", v.Reason)
	}
}

// A file git still lists but which is gone from the working tree is not a
// violation: a deletion is part of what the commit contains.
func TestInspect_IgnoresAFileAlreadyDeleted(t *testing.T) {
	guard := GuardStage{MaxFileSize: 1}

	violations, err := guard.Inspect(tree(), []string{"gone.py"}, []string{"anything"})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("a deleted file must not be reported, got %v", violations)
	}
}

func TestInspect_PassesACleanRepository(t *testing.T) {
	guard := GuardStage{
		ForbiddenPaths: []string{"*.ckpt", "data/raw/*"},
		MaxFileSize:    8192,
	}

	violations, err := guard.Inspect(tree(), []string{"src/model.py", "docs/notes.md"}, []string{"absent"})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("nothing should have been reported, got %v", violations)
	}
}

func TestForbiddenWords_SplitsAndNormalises(t *testing.T) {
	words := ForbiddenWords(" Alpha ,\n\nBETA\n , ")

	if len(words) != 2 || words[0] != "alpha" || words[1] != "beta" {
		t.Fatalf("want [alpha beta], got %v", words)
	}
}

func TestForbiddenWords_EmptyValueYieldsNothing(t *testing.T) {
	if words := ForbiddenWords(""); len(words) != 0 {
		t.Fatalf("want none, got %v", words)
	}
}
