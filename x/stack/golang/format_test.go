package golang

import (
	"context"
	"os"
	"path/filepath"
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

// goFile writes one file into a directory of its own and returns it.
func goFile(t *testing.T, name, content string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}
