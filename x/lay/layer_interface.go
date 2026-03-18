package lay

import (
	"context"
	"github.com/rridane/golang-pipeyard/pi"
)

type Layer[C any] interface {
	Name() string
	Run(ctx context.Context, pipeyard pi.Ctx, c C) error
}
