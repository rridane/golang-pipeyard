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

type PipelineConfig struct {
	Name             string
	ShortDescription string
	LongDescription  string
	Version          string
	Injectors        []any
	Assets           fs.FS
}

// RegisterStage panics on a stage whose Execute cannot be dispatched. Execute
// is absent from the Stage interface by design, so the compiler cannot check it.
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

// Run turns a runtime error into a non-zero exit status. The runtime has
// already reported it to the user.
func (p *Pipeline) Run(ctx context.Context) {
	if err := p.runtime.Process(ctx); err != nil {
		os.Exit(1)
	}
}

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
