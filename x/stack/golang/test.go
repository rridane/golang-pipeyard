package golang

import (
	"context"
	"fmt"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"go.uber.org/zap"
)

// DefaultTimeout is how long a test may run before the suite is declared
// stuck.
const DefaultTimeout = "5m"

// TestStage runs the suite on the host, with the race detector on by default:
// the bugs it finds are the ones that survive review.
type TestStage struct {
	// Packages is what --pkg falls back to. Empty uses DefaultPackages.
	Packages string

	// Timeout is what --timeout falls back to. Empty uses DefaultTimeout.
	Timeout string

	// WithoutRace turns --race off by default, for a toolchain with no C
	// compiler: the race detector needs cgo.
	WithoutRace bool
}

func (s *TestStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("test")
	spec.SetDescription("Run the test suite")
	spec.AddFlag("pkg", ru.StringFlag, "packages to test", s.packages(), false)
	spec.AddFlag("race", ru.BoolFlag, "enable the race detector", !s.WithoutRace, false)
	spec.AddFlag("timeout", ru.DurationFlag, "test timeout", s.timeout(), false)
	return spec
}

func (s *TestStage) Execute(
	ctx context.Context,
	args ru.ParsedArgs,
	sh *mod.ShellExecutor,
	log *zap.Logger,
) error {
	log.Info("running tests",
		zap.String("pkg", args.String("pkg")),
		zap.Bool("race", args.Bool("race")))

	return sh.Run(ctx, s.command(args))
}

// command is where the flags come together, kept out of Execute so that a
// test reaches it without running a go test inside a go test.
func (s *TestStage) command(args ru.ParsedArgs) string {
	flags := fmt.Sprintf("-timeout %s", args.Duration("timeout"))

	if args.Bool("race") {
		flags += " -race"
	}

	return fmt.Sprintf("go test %s %s", flags, args.String("pkg"))
}

func (s *TestStage) packages() string {
	if s.Packages == "" {
		return DefaultPackages
	}
	return s.Packages
}

func (s *TestStage) timeout() string {
	if s.Timeout == "" {
		return DefaultTimeout
	}
	return s.Timeout
}
