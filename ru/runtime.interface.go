package ru

import (
	"context"
)

type RuntimeInterface interface {
	AddCommand(cmd Command)
	Process(ctx context.Context) error
}
