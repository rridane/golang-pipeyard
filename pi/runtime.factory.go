package pi

import (
	"github.com/rridane/golang-pipeyard/pi/internal/runtime/cobra"
)

type RuntimeFactory struct {
	Name    string
	Short   string
	Long    string
	Version string
}

func (r *RuntimeFactory) NewRuntime() RuntimeInterface {
	return cobra.CobraRuntime{}
}
