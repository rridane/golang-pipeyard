package contracts

import (
	"context"
	"github.com/rridane/golang-pipeyard/internal"
)

type RuntimeInterface interface {
	AddCommand(cmd Command)
	Process(ctx context.Context) error
}

type Command struct {
	Command string
	Short   string
	Long    string
	Args    []FlagSpec
	Exec    func(ctx context.Context, args internal.ParsedArgs) error
}
