package layers

import (
	"context"
	"github.com/rridane/golang-pipeyard/pkg/pipeyard_ctx"
)

type Layer[C any] interface {
	Name() string
	Run(ctx context.Context, pipeyard pipeyard_ctx.PipeyardCtx, c C) error
}
