package golang

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"go.uber.org/zap"
)

// gofmt lists the files it would rewrite and exits zero whether there are any
// or none. A stage reading the exit code instead of the output would therefore
// pass on a repository that is not formatted, and pass for ever. These two
// tests are what hold that invariant: the first one fails the moment the stage
// stops reading what gofmt printed.
func TestFormatStage_FailsOnAnUnformattedFile(t *testing.T) {
	dir := goFile(t, "crooked.go", "package p\nfunc  f( )  {}\n")

	err := (&FormatStage{Paths: []string{"."}}).Execute(
		context.Background(), ru.ParsedArgs{}, &mod.ShellExecutor{Dir: dir}, zap.NewNop())

	if err == nil {
		t.Fatal("an unformatted file has to fail the stage")
	}
	if !strings.Contains(err.Error(), "crooked.go") {
		t.Fatalf("the error has to name the file to fix, got %v", err)
	}
}

func TestFormatStage_PassesOnAFormattedFile(t *testing.T) {
	dir := goFile(t, "straight.go", "package p\n\nfunc f() {}\n")

	err := (&FormatStage{Paths: []string{"."}}).Execute(
		context.Background(), ru.ParsedArgs{}, &mod.ShellExecutor{Dir: dir}, zap.NewNop())

	if err != nil {
		t.Fatalf("a formatted file has to pass, got %v", err)
	}
}

// A copy of the repository under out/ is what git ignores, and what gofmt
// would have walked into and counted twice.
func TestFormatStage_ChecksOnlyWhatGitSees(t *testing.T) {
	dir := goFile(t, "straight.go", "package p\n\nfunc f() {}\n")
	write(t, dir, ".gitignore", "out/\n")
	write(t, dir, "out/workspace/crooked.go", "package p\nfunc  f( )  {}\n")

	err := (&FormatStage{}).Execute(
		context.Background(), ru.ParsedArgs{}, &mod.ShellExecutor{Dir: dir}, zap.NewNop())

	if err != nil {
		t.Fatalf("an ignored copy must not count, got %v", err)
	}
}

func TestGoFiles_KeepsTheGoFilesUnderThePaths(t *testing.T) {
	files := []string{"pi/stage.go", "pi/doc.md", "x/stack/tool.go", "examples/main.go", "main.go"}

	if got := goFiles(files, []string{"pi", "x/"}); !slices.Equal(got, []string{"pi/stage.go", "x/stack/tool.go"}) {
		t.Errorf("got %v", got)
	}
	if got := goFiles(files, []string{"."}); len(got) != 4 {
		t.Errorf("the whole repository should keep every Go file, got %v", got)
	}
}

// goFile writes one file into a git repository of its own and returns it.
func goFile(t *testing.T, name, content string) string {
	t.Helper()

	dir := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	write(t, dir, name, content)
	return dir
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
