package inj

import (
	"github.com/rridane/golang-pipeyard/x/mod"
	"time"
)

type ShellExecutorInjector struct {
	timeout time.Duration
}

func (i *ShellExecutorInjector) Inject(stageName string) *mod.ShellExecutor {
	return &mod.ShellExecutor{
		Timeout: i.timeout,
	}
}
