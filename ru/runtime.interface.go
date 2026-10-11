package ru

import (
	"context"
)

type RuntimeInterface interface {
	AddCommand(cmd Command)

	// Process runs the CLI. It is responsible for reporting any error to the
	// user; the caller only turns a non-nil error into a non-zero exit status.
	Process(ctx context.Context) error
}
