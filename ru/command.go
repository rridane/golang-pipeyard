package ru

import "context"

type Command struct {
	Command string
	Short   string
	Long    string
	Args    []Flag
	Exec    func(ctx context.Context, args ParsedArgs) error
}
