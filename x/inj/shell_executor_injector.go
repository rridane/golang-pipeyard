package inj

import (
	"time"

	"github.com/rridane/golang-pipeyard/x/mod"
)

type ShellExecutorInjector struct {
	Timeout time.Duration // zero means no timeout
	Dir     string        // working directory; empty means the process one
}

func (i *ShellExecutorInjector) Inject(stageName string) *mod.ShellExecutor {
	return &mod.ShellExecutor{
		Timeout: i.Timeout,
		Dir:     i.Dir,
	}
}
