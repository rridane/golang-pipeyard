package core

import (
	"context"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
	"os"
	"strings"
	"time"
)

type ShellExecutor struct {
	Timeout time.Duration
}

func (s *ShellExecutor) Run(ctx context.Context, script string) error {
	file, _ := syntax.NewParser().Parse(strings.NewReader(script), "")
	runner, _ := interp.New(interp.StdIO(os.Stdin, os.Stdout, os.Stderr))
	_ = runner.Run(context.Background(), file)
	return nil
}
