package runtime

import (
	"github.com/rridane/golang-pipeyard/x"
)

type RuntimeFactory struct {
	Name    string
	Short   string
	Long    string
	Version string
}

func (r *RuntimeFactory) NewRuntime() x.RuntimeInterface {
	return CobraRuntime{}
}
