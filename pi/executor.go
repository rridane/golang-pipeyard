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
	injectorMethods []reflect.Value
	Observer        StageObserver
}

func (e *executor) generateExec(stage Stage, stageName string) func(ctx context.Context, args ru.ParsedArgs) error {

	return func(ctx context.Context, args ru.ParsedArgs) (err error) {

		ctx, finished := e.observe(ctx, stageName, args)

		// Deferred, so a stage that panics is still recorded as a failure.
		// Called explicitly it would leave the span open and the counter
		// short, and a panicking deploy would not appear in the failure rate.
		defer func() {
			if recovered := recover(); recovered != nil {
				finished(fmt.Errorf("stage %q panicked: %v", stageName, recovered))
				panic(recovered)
			}
			finished(err)
		}()

		err = e.run(ctx, stage, stageName, args)

		return err
	}
}

func (e *executor) observe(ctx context.Context, stageName string, args ru.ParsedArgs) (context.Context, StageFinished) {
	if e.Observer == nil {
		return ctx, func(error) {}
	}
	return e.Observer.StageStarted(ctx, stageName, args)
}

func (e *executor) run(ctx context.Context, stage Stage, stageName string, args ru.ParsedArgs) error {

	signature, found := getExecuteMethodSignature(stage)

	if !found {
		return fmt.Errorf("stage %q: no Execute method", stageName)
	}

	dependencies, err := dependency_injection.GetDependencies(
		signature, e.injectorMethods, stageName)

	if err != nil {
		return err
	}

	funcArgs := append([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(args),
	}, dependencies...)

	return callExecute(getExecuteMethod(stage), funcArgs, stageName)
}

func callExecute(method reflect.Value, args []reflect.Value, stageName string) error {

	results := method.Call(args)

	if len(results) == 0 {
		return fmt.Errorf("stage %q: Execute returned no value, expected error", stageName)
	}

	last := results[len(results)-1]

	// A concrete error type is accepted here because checkStage accepts it at
	// registration: a stage that registers has to be able to run. Reading
	// IsNil first also turns a nil *MyError into no error, which converting to
	// the interface would not.
	switch last.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		if last.IsNil() {
			return nil
		}
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
