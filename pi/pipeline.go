package pi

import (
	"context"
	"fmt"
	"github.com/rridane/golang-pipeyard/pi/ru"
	"os"
)

type Pipeline struct {
	runtime   ru.RuntimeInterface
	injectors []any
	executor  executor
}

// PipelineConfig configure l'application
type PipelineConfig struct {
	Name             string
	ShortDescription string
	LongDescription  string
	Version          string
	Injectors        []any
}

// RegisterStage enregistre un stage
func (p *Pipeline) RegisterStage(stage Stage) *Pipeline {

	stageSpec := stage.Configure()

	commandSpec := ru.Command{
		Command: stageSpec.Command,
		Short:   stageSpec.Short,
		Long:    stageSpec.Long,
		Args:    stageSpec.Args,
		Exec:    p.executor.generateExec(stage, stageSpec.Command),
	}

	p.runtime.AddCommand(commandSpec)

	return p
}

func (p *Pipeline) Run(ctx context.Context) {
	if err := p.runtime.Process(ctx); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "❌ Erreur: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
}

// NewPipeline crée un nouveau pipeline
func NewPipeline(config PipelineConfig) *Pipeline {

	runtimeFactory := RuntimeFactory{
		Name:    config.Name,
		Short:   config.ShortDescription,
		Long:    config.LongDescription,
		Version: config.Version,
	}

	r := runtimeFactory.NewRuntime()
	e := executor{Injectors: config.Injectors}

	app := &Pipeline{
		runtime:   r,
		injectors: config.Injectors,
		executor:  e,
	}

	return app
}
