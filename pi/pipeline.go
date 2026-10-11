package pi

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/rridane/golang-pipeyard/pi/internal/dependency_injection"
	"github.com/rridane/golang-pipeyard/ru"
)

type Pipeline struct {
	runtime  ru.RuntimeInterface
	executor executor
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
	if err := p.process(ctx); err != nil {
		os.Exit(1)
	}
}

// process flushes the observer whichever way the runtime leaves. A panic skips
// the rest of Run and os.Exit skips deferred calls, and an observer that is
// never flushed records nothing.
func (p *Pipeline) process(ctx context.Context) error {
	defer p.flush(ctx)
	return p.runtime.Process(ctx)
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

// injectorMethods panics on a malformed injector, like RegisterStage does on a
// malformed stage: a miswired pipeline must not start and then fail inside a
// stage on an error that names the wrong thing.
func injectorMethods(injectors []any) []reflect.Value {

	methods := make([]reflect.Value, 0, len(injectors))

	for _, inj := range injectors {
		method, err := dependency_injection.ValidateDependencyInjector(inj)
		if err != nil {
			panic(err)
		}
		methods = append(methods, method)
	}

	return methods
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

	e := executor{injectorMethods: injectorMethods(injectors), Observer: config.Observer}

	return &Pipeline{
		runtime:  r,
		executor: e,
	}
}
