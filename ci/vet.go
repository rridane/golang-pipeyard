package main

import (
	"context"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"go.uber.org/zap"
)

type VetStage struct{}

func (s *VetStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("vet")
	spec.SetDescription("Run go vet on every package")
	return spec
}

func (s *VetStage) Execute(ctx context.Context, args ru.ParsedArgs, sh *mod.ShellExecutor, log *zap.Logger) error {
	if err := sh.Run(ctx, "go vet ./..."); err != nil {
		return err
	}

	log.Info("vet found nothing")
	return nil
}
