package adapters

import (
	"github.com/rridane/golang-pipeyard/internal/contracts"
)

type RuntimeFactory struct {
	Name    string
	Short   string
	Long    string
	Version string
}

func (r *RuntimeFactory) NewRuntime() contracts.RuntimeInterface {
	return CobraRuntime{}
}
