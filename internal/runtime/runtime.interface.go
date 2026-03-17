package runtime

import (
	"context"
)

type RuntimeInterface interface {
	AddCommand(cmd Command)
	Process(ctx context.Context) error
}

type Command struct {
	Command string
	Short   string
	Long    string
	Args    []Flag
	Exec    func(ctx context.Context, args ParsedArgsInterface) error
}
