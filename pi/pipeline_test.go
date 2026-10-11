package pi

import (
	"context"
	"testing"

	"github.com/rridane/golang-pipeyard/ru"
)

type panickingRuntime struct{}

func (r *panickingRuntime) AddCommand(ru.Command)         {}
func (r *panickingRuntime) Process(context.Context) error { panic("stage exploded") }

type closingObserver struct {
	recordingObserver
	closed int
}

func (o *closingObserver) Close(context.Context) error {
	o.closed++
	return nil
}

// A batching exporter holds the panicking stage's span until something flushes
// it, and Run is the only place that can.
func TestRun_FlushesWhenAStagePanics(t *testing.T) {
	observer := &closingObserver{}
	p := &Pipeline{runtime: &panickingRuntime{}, executor: executor{Observer: observer}}

	func() {
		defer func() {
			if recover() == nil {
				t.Error("the panic must keep propagating")
			}
		}()
		p.Run(context.Background())
	}()

	if observer.closed != 1 {
		t.Fatalf("want the observer flushed once, got %d", observer.closed)
	}
}

type brokenInjector struct{}

func (i *brokenInjector) Inject() string { return "" }

func TestNewPipeline_RejectsAMalformedInjector(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("a malformed injector must stop the pipeline from starting")
		}
	}()
	NewPipeline(PipelineConfig{Name: "p", Injectors: []any{&brokenInjector{}}})
}

type identityStage struct{ seen Identity }

func (s *identityStage) Configure() *StageSpec { return NewStageSpec("identity") }

func (s *identityStage) Execute(ctx context.Context, args ru.ParsedArgs, run Identity) error {
	s.seen = run
	return nil
}

type runInjector struct{}

func (i *runInjector) Inject(stageName string) Identity { return ReadIdentity() }

// A stage that needs the commit it is building declares it, instead of reading
// the environment itself. sys.IdentityInjector is the injector that serves it.
func TestIdentity_IsInjectableIntoAStage(t *testing.T) {
	t.Setenv(ProjectEnvVar, "acme/widgets")
	t.Setenv(BranchEnvVar, "main")

	stage := &identityStage{}
	e := executor{injectorMethods: injectorMethods([]any{&runInjector{}})}

	if err := e.generateExec(stage, "identity")(context.Background(), ru.ParsedArgs{}); err != nil {
		t.Fatal(err)
	}

	if stage.seen.Project != "acme/widgets" || stage.seen.Branch != "main" {
		t.Fatalf("the stage must receive the run identity, got %+v", stage.seen)
	}
}
