package injectors

import (
	shell_executor "github.com/rridane/golang-pipeyard/pkg/shell_executor"
	"time"
)

type ShellExecutorInjector struct {
	timeout time.Duration
}

func (i *ShellExecutorInjector) Inject(stageName string) *shell_executor.ShellExecutor {
	return &shell_executor.ShellExecutor{
		Timeout: i.timeout,
	}
}
