package pi

import (
	"context"
	"fmt"
	"reflect"

	"github.com/rridane/golang-pipeyard/pi/internal/dependency_injection"
	"github.com/rridane/golang-pipeyard/ru"
)

// Execute is not part of the Stage interface, so the compiler cannot check it.
// These type tokens let the executor do that check itself.
var (
	errorType      = reflect.TypeOf((*error)(nil)).Elem()
	contextType    = reflect.TypeOf((*context.Context)(nil)).Elem()
	parsedArgsType = reflect.TypeOf(ru.ParsedArgs{})
)

type executor struct {
	Injectors []any
}

func (e *executor) generateExec(stage Stage, stageName string) func(ctx context.Context, args ru.ParsedArgs) error {

	return func(ctx context.Context, args ru.ParsedArgs) error {

		signature, found := getExecuteMethodSignature(stage)

		if !found {
			return fmt.Errorf("stage %q: no Execute method", stageName)
		}

		injectorMethods := e.validateInjectorAndGetMethods()

		dependencies, err := dependency_injection.GetDependencies(signature, injectorMethods, stageName)

		if err != nil {
			return err
		}

		funcArgs := append([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(args),
		}, dependencies...)

		return callExecute(getExecuteMethod(stage), funcArgs, stageName)
	}
}

func callExecute(method reflect.Value, args []reflect.Value, stageName string) error {

	results := method.Call(args)

	if len(results) == 0 {
		return fmt.Errorf("stage %q: Execute returned no value, expected error", stageName)
	}

	last := results[len(results)-1]

	if last.Kind() != reflect.Interface {
		return fmt.Errorf("stage %q: Execute must return error, got %s", stageName, last.Type())
	}

	if last.IsNil() {
		return nil
	}

	failure, ok := last.Interface().(error)

	if !ok {
		return fmt.Errorf("stage %q: Execute must return error, got %s", stageName, last.Type())
	}

	return failure
}

// checkStage runs at registration: a pipeline that cannot report failure must
// not start at all.
func (e *executor) checkStage(stage Stage, stageName string) error {

	signature, found := getExecuteMethodSignature(stage)

	if !found {
		return fmt.Errorf(
			"stage %q: no Execute method; expected "+
				"Execute(ctx context.Context, args ru.ParsedArgs, deps ...) error", stageName)
	}

	// 0 receiver, 1 context, 2 parsed args, 3+ dependencies
	if signature.Type.NumIn() < 3 {
		return fmt.Errorf(
			"stage %q: Execute must take at least (ctx context.Context, args ru.ParsedArgs)", stageName)
	}

	if signature.Type.In(1) != contextType {
		return fmt.Errorf("stage %q: first Execute parameter must be context.Context, got %s",
			stageName, signature.Type.In(1))
	}

	if signature.Type.In(2) != parsedArgsType {
		return fmt.Errorf("stage %q: second Execute parameter must be ru.ParsedArgs, got %s",
			stageName, signature.Type.In(2))
	}

	if signature.Type.NumOut() != 1 {
		return fmt.Errorf("stage %q: Execute must return exactly one error, got %d return values",
			stageName, signature.Type.NumOut())
	}

	if !signature.Type.Out(0).Implements(errorType) {
		return fmt.Errorf("stage %q: Execute must return error, got %s",
			stageName, signature.Type.Out(0))
	}

	return nil
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

func getExecuteMethod(stage Stage) reflect.Value {
	v := reflect.ValueOf(stage)
	meth := v.MethodByName("Execute")
	return meth
}
