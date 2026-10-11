package pi

import (
	"context"

	"github.com/rridane/golang-pipeyard/ru"
)

// StageObserver is notified around every stage run. A pipeline without one is
// not instrumented; wiring one instruments every stage, with no code in the
// stages themselves and no way for a stage to opt out.
type StageObserver interface {
	StageStarted(ctx context.Context, stage string, args ru.ParsedArgs) (context.Context, StageFinished)
}

// StageFinished is called exactly once, with the stage's error or nil.
type StageFinished func(error)

// StageObserverCloser is flushed by Run before the process exits. A batching
// exporter in a short-lived command drops everything otherwise.
type StageObserverCloser interface {
	Close(ctx context.Context) error
}
