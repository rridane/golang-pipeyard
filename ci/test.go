package main

import (
	"context"
	"fmt"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"go.uber.org/zap"
)

type TestStage struct{}

func (s *TestStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("test")
	spec.SetDescription("Run the test suite")
	spec.AddFlag("pkg", ru.StringFlag, "packages to test", "./...", false)
	spec.AddFlag("race", ru.BoolFlag, "enable the race detector", true, false)
	spec.AddFlag("timeout", ru.DurationFlag, "test timeout", "5m", false)
	return spec
}

func (s *TestStage) Execute(ctx context.Context, args ru.ParsedArgs, sh *mod.ShellExecutor, log *zap.Logger) error {
	log.Info("running tests",
		zap.String("pkg", args.String("pkg")),
		zap.Bool("race", args.Bool("race")))

	return sh.Run(ctx, testScript(args))
}

// testScript is where the flags come together, kept out of Execute so a test
// can check it without running a go test inside a go test.
func testScript(args ru.ParsedArgs) string {
	flags := fmt.Sprintf("-timeout %s", args.Duration("timeout"))

	if args.Bool("race") {
		flags += " -race"
	}

	return fmt.Sprintf("go test %s %s", flags, args.String("pkg"))
}
