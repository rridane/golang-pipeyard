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

// StageObserverCloser is implemented by an observer that buffers what it
// records. Run flushes it before the process exits: a batching exporter in a
// short-lived command drops everything otherwise, so an unflushed pipeline
// produces telemetry and throws it away.
type StageObserverCloser interface {
	Close(ctx context.Context) error
}
