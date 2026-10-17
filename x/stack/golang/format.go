package golang

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"github.com/rridane/golang-pipeyard/x/stack"
	"go.uber.org/zap"
)

// FormatStage reports the files gofmt would rewrite, among the Go files git
// sees. gofmt is handed files, never a directory: walking one, it would read a
// workspace copy of the repository under out/ and stop on the first directory
// it cannot open.
type FormatStage struct {
	// Paths are the directories to examine, relative to the repository
	// root. Empty examines the whole repository.
	Paths []string
}

func (s *FormatStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("fmt")
	spec.SetDescription("Check that every file is gofmt'd")
	return spec
}

func (s *FormatStage) Execute(
	ctx context.Context,
	args ru.ParsedArgs,
	sh *mod.ShellExecutor,
	log *zap.Logger,
) error {
	paths := s.Paths
	if len(paths) == 0 {
		paths = []string{"."}
	}

	tracked, err := stack.TrackedFiles(ctx, sh)
	if err != nil {
		return err
	}
	files := goFiles(tracked, paths)
	if len(files) == 0 {
		log.Info("no Go file to check", zap.Strings("paths", paths))
		return nil
	}

	out, err := sh.Output(ctx, s.command(files))
	if err != nil {
		return fmt.Errorf("gofmt: %w", err)
	}

	unformatted := strings.Fields(string(out))

	if len(unformatted) > 0 {
		return fmt.Errorf("%d file(s) need gofmt: %s",
			len(unformatted), strings.Join(unformatted, " "))
	}

	log.Info("every package is formatted", zap.Strings("paths", paths))
	return nil
}

func (s *FormatStage) command(files []string) string {
	return "gofmt -l " + stack.QuoteAll(files)
}

// goFiles keeps the Go files under one of the paths.
func goFiles(files, paths []string) []string {
	var kept []string
	for _, file := range files {
		if !strings.HasSuffix(file, ".go") {
			continue
		}
		for _, dir := range paths {
			dir = strings.TrimSuffix(path.Clean(dir), "/")
			if dir == "." || file == dir || strings.HasPrefix(file, dir+"/") {
				kept = append(kept, file)
				break
			}
		}
	}
	return kept
}
