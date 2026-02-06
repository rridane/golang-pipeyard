package pipeyard

import (
	"context"
	"fmt"
	"github.com/rridane/golang-pipeyard/contracts"
	"github.com/rridane/golang-pipeyard/internal"
	"github.com/rridane/golang-pipeyard/internal/adapters"
	"github.com/rridane/golang-pipeyard/internal/dependency_injection"
	"os"
	"reflect"
)

type Pipeline struct {
	runtime   contracts.RuntimeInterface
	injectors []any
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
func (a *Pipeline) RegisterStage(stage Stage) *Pipeline {

	stageSpec := stage.Configure()

	commandSpec := contracts.Command{
		Command: stageSpec.Command,
		Short:   stageSpec.Short,
		Long:    stageSpec.Long,
		Args:    stageSpec.Args,
		Exec:    a.generateCommandRun(stage, stageSpec.Command),
	}

	a.runtime.AddCommand(commandSpec)
	return a
}

func (a *Pipeline) generateCommandRun(stage Stage, stageName string) func(ctx context.Context, args internal.ParsedArgs) error {

	return func(ctx context.Context, args internal.ParsedArgs) error {

		var injectorMethods []reflect.Value

		injectorMethods = a.validateInjectorAndGetMethods()

		executeMethodSignature, found := getExecuteMethodSignature(stage)

		if !found {
			return nil
		}

		funcArgs := []reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(args),
		}

		dependencies, err := dependency_injection.GetDependencies(executeMethodSignature, injectorMethods, stageName)

		if err != nil {
			return err
		}

		funcArgs = append(funcArgs, dependencies...)

		meth := getExecuteMethod(stage)

		meth.Call(funcArgs)

		return nil
	}
}

func (a *Pipeline) validateInjectorAndGetMethods() []reflect.Value {

	var injectorMethods []reflect.Value

	for i := range a.injectors {

		inj := a.injectors[i]
		method, err := dependency_injection.ValidateDependencyInjector(inj)
		if err != nil {
			fmt.Println("❌", err)
			continue
		}

		injectorMethods = append(injectorMethods, method)
	}
	return injectorMethods
}

func (a *Pipeline) Run(ctx context.Context) {
	if err := a.runtime.Process(ctx); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "❌ Erreur: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
}

// NewPipeline crée un nouveau pipeline
func NewPipeline(config PipelineConfig) *Pipeline {

	runtimeFactory := adapters.RuntimeFactory{
		Name:    config.Name,
		Short:   config.ShortDescription,
		Long:    config.LongDescription,
		Version: config.Version,
	}

	runtime := runtimeFactory.NewRuntime()

	app := &Pipeline{
		runtime:   runtime,
		injectors: config.Injectors,
	}

	return app
}
