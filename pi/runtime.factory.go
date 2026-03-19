package pi

import (
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/ru/cobra"
)

type RuntimeFactory struct {
	Name    string
	Short   string
	Long    string
	Version string
}

func (r *RuntimeFactory) NewRuntime() ru.RuntimeInterface {
	return &cobra.CobraRuntime{}
}
