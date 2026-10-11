package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"go.uber.org/zap"
)

type FormatStage struct{}

func (s *FormatStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("fmt")
	spec.SetDescription("Check that every package is gofmt'd")
	return spec
}

func (s *FormatStage) Execute(ctx context.Context, args ru.ParsedArgs, sh *mod.ShellExecutor, log *zap.Logger) error {
	out, err := sh.Output(ctx, "gofmt -l pi ru x ci")
	if err != nil {
		return fmt.Errorf("gofmt: %w", err)
	}

	unformatted := strings.Fields(string(out))
	if len(unformatted) > 0 {
		return fmt.Errorf("%d file(s) need gofmt: %s", len(unformatted), strings.Join(unformatted, " "))
	}

	log.Info("every package is formatted")
	return nil
}
