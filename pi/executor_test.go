package pi

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rridane/golang-pipeyard/ru"
)

type failingStage struct{}

func (s *failingStage) Configure() *StageSpec { return NewStageSpec("fail") }
func (s *failingStage) Execute(ctx context.Context, args ru.ParsedArgs) error {
	return errors.New("tests failed: 3 assertions")
}

type succeedingStage struct{}

func (s *succeedingStage) Configure() *StageSpec { return NewStageSpec("ok") }
func (s *succeedingStage) Execute(ctx context.Context, args ru.ParsedArgs) error {
	return nil
}

// A stage that fails must fail the command. Before this was enforced, the
// executor discarded Execute's return value and every pipeline exited zero.
func TestExec_StageFailureReachesTheCaller(t *testing.T) {
	e := executor{}

	err := e.generateExec(&failingStage{}, "fail")(context.Background(), ru.ParsedArgs{})

	if err == nil {
		t.Fatal("a stage returning an error must fail the command")
	}
	if !strings.Contains(err.Error(), "3 assertions") {
		t.Fatalf("the stage's own error must reach the caller, got: %v", err)
	}
}

func TestExec_StageSuccessReturnsNil(t *testing.T) {
	e := executor{}

	if err := e.generateExec(&succeedingStage{}, "ok")(context.Background(), ru.ParsedArgs{}); err != nil {
		t.Fatalf("a stage returning nil must succeed, got: %v", err)
	}
}

type missingExecuteStage struct{}

func (s *missingExecuteStage) Configure() *StageSpec { return NewStageSpec("typo") }

// lowercase: the compiler accepts it, reflection does not find it
func (s *missingExecuteStage) execute(ctx context.Context, args ru.ParsedArgs) error { return nil }

type noReturnStage struct{}

func (s *noReturnStage) Configure() *StageSpec                           { return NewStageSpec("noret") }
func (s *noReturnStage) Execute(ctx context.Context, args ru.ParsedArgs) {}

type wrongReturnStage struct{}

func (s *wrongReturnStage) Configure() *StageSpec                               { return NewStageSpec("badret") }
func (s *wrongReturnStage) Execute(ctx context.Context, args ru.ParsedArgs) int { return 0 }

type wrongParamsStage struct{}

func (s *wrongParamsStage) Configure() *StageSpec { return NewStageSpec("badparams") }
func (s *wrongParamsStage) Execute(args ru.ParsedArgs, ctx context.Context) error {
	return nil
}

// Execute is absent from the Stage interface, so the compiler cannot check it.
// Registration is the only place left, and a stage that cannot report failure
// must not reach the CI engine.
func TestCheckStage_RejectsStagesThatCannotReportFailure(t *testing.T) {
	cases := map[string]Stage{
		"no Execute method":       &missingExecuteStage{},
		"no return value":         &noReturnStage{},
		"return is not error":     &wrongReturnStage{},
		"parameters out of order": &wrongParamsStage{},
	}

	e := executor{}

	for name, stage := range cases {
		t.Run(name, func(t *testing.T) {
			if err := e.checkStage(stage, stage.Configure().command); err == nil {
				t.Fatal("this stage must be rejected at registration")
			}
		})
	}
}

func TestCheckStage_AcceptsAWellFormedStage(t *testing.T) {
	e := executor{}

	if err := e.checkStage(&succeedingStage{}, "ok"); err != nil {
		t.Fatalf("a well-formed stage must be accepted, got: %v", err)
	}
}
