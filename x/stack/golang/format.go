package golang

import (
	"context"
	"fmt"
	"strings"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"github.com/rridane/golang-pipeyard/x/stack"
	"go.uber.org/zap"
)

// FormatStage reports the files gofmt would rewrite.
type FormatStage struct {
	// Paths are the directories to examine, relative to the repository
	// root. Empty examines the whole repository.
	//
	// gofmt walks whatever it is given and stops on the first directory it cannot
	// open, even one holding no Go at all.
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

	out, err := sh.Output(ctx, s.command(paths))
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

func (s *FormatStage) command(paths []string) string {
	return "gofmt -l " + stack.QuoteAll(paths)
}
