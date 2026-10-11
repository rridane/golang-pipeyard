package dependency_injection

import (
	"fmt"
	"reflect"
)

func GetDependencies(m reflect.Method, injectMethods []reflect.Value, stageName string) ([]reflect.Value, error) {

	var arg []reflect.Value
	var missingDeps []string
	numParams := m.Type.NumIn()

	// 1: context.Context
	// 2: *parsedArgs
	// 3+: dependencies
	for p := 3; p < numParams; p++ {

		indexPParamType := m.Type.In(p)
		found := false

		for i := 0; i < len(injectMethods); i++ {

			injectorMethod := injectMethods[i]
			returnType := injectorMethod.Type().Out(0)

			if !returnType.AssignableTo(indexPParamType) {
				continue
			}

			injectResponse := injectorMethod.Call([]reflect.Value{reflect.ValueOf(stageName)})

			if len(injectResponse) == 0 {
				continue
			}

			injectedService := injectResponse[0]

			// An injector signals "not configured" with nil, which only a nilable
			// kind can do. IsNil on anything else panics.
			switch injectedService.Kind() {
			case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
				if injectedService.IsNil() {
					continue
				}
			}

			if injectedService.Type().AssignableTo(indexPParamType) {
				arg = append(arg, injectedService)
				found = true
				injectMethods = append(injectMethods[:i], injectMethods[i+1:]...)
				i--
				break
			}
		}

		if !found {
			missingDeps = append(missingDeps, indexPParamType.String())
		}

	}

	if len(missingDeps) > 0 {
		return nil, fmt.Errorf(
			"stage %q: no injector provided %v; either it is not registered in "+
				"PipelineConfig.Injectors, or its Inject returned nil because its "+
				"configuration is incomplete", stageName, missingDeps)
	}

	return arg, nil
}

func ValidateDependencyInjector(inj any) (reflect.Value, error) {

	injType := reflect.TypeOf(inj)
	injValue := reflect.ValueOf(inj)

	method, ok := injType.MethodByName("Inject")

	if !ok {
		return reflect.Value{}, fmt.Errorf("%T has no Inject method", inj)
	}

	if method.Type.NumIn() != 2 {
		return reflect.Value{}, fmt.Errorf("%T.Inject must take exactly one parameter (string)", inj)
	}

	paramType := method.Type.In(1)
	if paramType.Kind() != reflect.String {
		return reflect.Value{}, fmt.Errorf("%T.Inject must take a string (got %v)", inj, paramType)
	}

	if method.Type.NumOut() != 1 {
		return reflect.Value{}, fmt.Errorf("%T.Inject must return exactly one value", inj)
	}

	injectMethod := injValue.MethodByName("Inject")

	if !injectMethod.IsValid() {
		return reflect.Value{}, fmt.Errorf("%T.Inject not found on the value", inj)
	}

	return injectMethod, nil
}
