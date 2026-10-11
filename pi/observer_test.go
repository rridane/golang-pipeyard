package pi

import (
	"context"
	"strings"
	"testing"

	"github.com/rridane/golang-pipeyard/ru"
)

type recordingObserver struct {
	started  int
	finished int
	err      error
}

func (o *recordingObserver) StageStarted(ctx context.Context, stage string, args ru.ParsedArgs) (context.Context, StageFinished) {
	o.started++
	return ctx, func(err error) {
		o.finished++
		o.err = err
	}
}

type panickingStage struct{}

func (s *panickingStage) Configure() *StageSpec { return NewStageSpec("panics") }
func (s *panickingStage) Execute(ctx context.Context, args ru.ParsedArgs) error {
	var m map[string]string
	m["boom"] = "x"
	return nil
}

func TestObserver_RecordsSuccess(t *testing.T) {
	observer := &recordingObserver{}
	e := executor{Observer: observer}

	if err := e.generateExec(&succeedingStage{}, "ok")(context.Background(), ru.ParsedArgs{}); err != nil {
		t.Fatal(err)
	}

	if observer.started != 1 || observer.finished != 1 {
		t.Fatalf("want one start and one finish, got %d and %d", observer.started, observer.finished)
	}
	if observer.err != nil {
		t.Fatalf("a successful stage must be recorded with no error, got %v", observer.err)
	}
}

func TestObserver_RecordsFailure(t *testing.T) {
	observer := &recordingObserver{}
	e := executor{Observer: observer}

	_ = e.generateExec(&failingStage{}, "fail")(context.Background(), ru.ParsedArgs{})

	if observer.err == nil {
		t.Fatal("a failing stage must be recorded with its error")
	}
}

// A stage that panics must still land in the metrics: an unrecorded failure
// reads as a lower deployment count, not as an error.
func TestObserver_RecordsAPanic(t *testing.T) {
	observer := &recordingObserver{}
	e := executor{Observer: observer}

	func() {
		defer func() {
			if recover() == nil {
				t.Error("the panic must keep propagating")
			}
		}()
		_ = e.generateExec(&panickingStage{}, "panics")(context.Background(), ru.ParsedArgs{})
	}()

	if observer.finished != 1 {
		t.Fatalf("the observer must be told once, got %d", observer.finished)
	}
	if observer.err == nil || !strings.Contains(observer.err.Error(), "panicked") {
		t.Fatalf("the panic must be recorded as a failure, got %v", observer.err)
	}
}

func TestObserver_AbsentObserverIsHarmless(t *testing.T) {
	e := executor{}

	if err := e.generateExec(&failingStage{}, "fail")(context.Background(), ru.ParsedArgs{}); err == nil {
		t.Fatal("a pipeline without an observer must still report failures")
	}
}
