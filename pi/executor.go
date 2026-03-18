package pi

import (
	"context"
	"fmt"
	"github.com/rridane/golang-pipeyard/pi/internal/dependency_injection"
	"github.com/rridane/golang-pipeyard/pi/ru"
	"reflect"
)

type executor struct {
	Injectors []any
}

func (e *executor) generateExec(stage Stage, stageName string) func(ctx context.Context, args ru.ParsedArgs) error {

	return func(ctx context.Context, args ru.ParsedArgs) error {

		var injectorMethods []reflect.Value

		injectorMethods = e.validateInjectorAndGetMethods()

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

func (e *executor) validateInjectorAndGetMethods() []reflect.Value {

	var injectorMethods []reflect.Value

	for i := range e.Injectors {

		inj := e.Injectors[i]
		method, err := dependency_injection.ValidateDependencyInjector(inj)
		if err != nil {
			fmt.Println("❌", err)
			continue
		}

		injectorMethods = append(injectorMethods, method)
	}
	return injectorMethods
}

func getExecuteMethodSignature(stage Stage) (reflect.Method, bool) {
	stageVal := reflect.TypeOf(stage)
	m, ok := stageVal.MethodByName("Execute")
	return m, ok
}

func getConfigureMethodSignature(stage Stage) (reflect.Method, bool) {
	stageVal := reflect.TypeOf(stage)
	m, ok := stageVal.MethodByName("Configure")
	return m, ok
}

func getExecuteMethod(stage Stage) reflect.Value {
	v := reflect.ValueOf(stage)
	meth := v.MethodByName("Execute")
	return meth
}
