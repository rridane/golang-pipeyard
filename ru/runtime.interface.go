package ru

import (
	"context"
)

type RuntimeInterface interface {
	AddCommand(cmd Command)

	// Process reports any error to the user; the caller only turns it into an
	// exit status.
	Process(ctx context.Context) error
}
