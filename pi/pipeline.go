package pi

import (
	"context"
	"fmt"
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

	// Observer instruments every stage. Leave it nil and nothing is recorded.
	Observer StageObserver
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
	err := p.runtime.Process(ctx)

	// Before either exit: os.Exit skips deferred calls, and a pipeline that
	// does not flush its observer records nothing.
	p.flush(ctx)

	if err != nil {
		os.Exit(1)
	}
}

func (p *Pipeline) flush(ctx context.Context) {
	closer, ok := p.executor.Observer.(StageObserverCloser)
	if !ok {
		return
	}
	if err := closer.Close(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "warning: observer not flushed: %v\n", err)
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

	// The pipeline context is injectable like anything else, so a stage that
	// needs the mount or working directory just declares it.
	injectors := append([]any{&ctxInjector{ctx: newCtx()}}, config.Injectors...)

	e := executor{Injectors: injectors, Observer: config.Observer}

	pipe := &Pipeline{
		runtime:   r,
		injectors: injectors,
		executor:  e,
	}

	return pipe
}
