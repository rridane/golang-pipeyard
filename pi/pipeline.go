package pi

import (
	"context"
	"io/fs"
	"os"

	"github.com/rridane/golang-pipeyard/ru"
)

type Pipeline struct {
	runtime   ru.RuntimeInterface
	injectors []any
	executor  executor
}

// PipelineConfig configures the application.
type PipelineConfig struct {
	Name             string
	ShortDescription string
	LongDescription  string
	Version          string
	Injectors        []any
	Assets           fs.FS
}

// RegisterStage adds a stage to the pipeline.
//
// It panics when the stage's Execute method cannot be dispatched. Execute is
// deliberately absent from the Stage interface, so the compiler cannot catch a
// wrong signature; registration is the only place left to catch it, and a
// pipeline that cannot report failure must not start at all.
func (p *Pipeline) RegisterStage(stage Stage) *Pipeline {

	stageSpec := stage.Configure()

	if err := p.executor.checkStage(stage, stageSpec.command); err != nil {
		panic(err)
	}

	commandSpec := ru.Command{
		Command: stageSpec.command,
		Short:   stageSpec.short,
		Long:    stageSpec.long,
		Args:    stageSpec.args,
		Exec:    p.executor.generateExec(stage, stageSpec.command),
	}

	p.runtime.AddCommand(commandSpec)

	return p
}

// Run executes the CLI. The runtime reports any error to the user; Run only
// turns it into a non-zero exit status, which is what the CI engine reads.
func (p *Pipeline) Run(ctx context.Context) {
	if err := p.runtime.Process(ctx); err != nil {
		os.Exit(1)
	}
}

// NewPipeline creates a new pipeline.
func NewPipeline(config PipelineConfig) *Pipeline {

	runtimeFactory := RuntimeFactory{
		Name:    config.Name,
		Short:   config.ShortDescription,
		Long:    config.LongDescription,
		Version: config.Version,
	}

	r := runtimeFactory.NewRuntime()
	e := executor{Injectors: config.Injectors}

	pipe := &Pipeline{
		runtime:   r,
		injectors: config.Injectors,
		executor:  e,
	}

	return pipe
}
