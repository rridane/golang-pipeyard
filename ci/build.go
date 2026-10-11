package main

import (
	"context"
	"fmt"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"go.uber.org/zap"
)

type BuildStage struct{}

func buildScript(args ru.ParsedArgs) string {
	return fmt.Sprintf("go build %s", args.String("pkg"))
}

func (s *BuildStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("build")
	spec.SetDescription("Compile every package")
	spec.AddFlag("pkg", ru.StringFlag, "packages to compile", "./...", false)
	return spec
}

// Execute compiles more than the pipeline does: ci/ imports neither the
// Kubernetes, Terraform, forge, notify nor feature-flag injectors, so without
// this a broken one of those would only surface in vet.
func (s *BuildStage) Execute(ctx context.Context, args ru.ParsedArgs,
	sh *mod.ShellExecutor, log *zap.Logger) error {

	if err := sh.Run(ctx, buildScript(args)); err != nil {
		return err
	}

	log.Info("every package compiles")
	return nil
}
