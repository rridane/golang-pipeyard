package pipeline_provider

import (
	"fmt"
	"reflect"
)

type DependencyInjector[T any] interface {
	Inject(stageName string) *T
}

// return serviceA reflect value for a method Method(serviceA) if injector of ServiceA is active
// Method ...params
func getDependencies(m reflect.Method, injectMethods []reflect.Value, stageName string) ([]reflect.Value, error) {
	var arg []reflect.Value
	var missingDeps []string
	numParams := m.Type.NumIn()

	// 1: context.Context
	// 2: *cobra.Command
	// 3+: dépendances
	for p := 3; p < numParams; p++ {

		indexPParamType := m.Type.In(p)
		found := false

		if indexPParamType.Kind() != reflect.Ptr {
			continue
		}

		for i := 0; i < len(injectMethods); i++ {
			injectorMethod := injectMethods[i]
			injectResponse := injectorMethod.Call([]reflect.Value{reflect.ValueOf(stageName)})

			if len(injectResponse) == 0 || injectResponse[0].IsNil() {
				continue
			}

			injectedService := injectResponse[0]

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
		return nil, fmt.Errorf("dépendances manquantes: %v", missingDeps)
	}

	return arg, nil
}

func (a *Pipeline) validateInjectorAndGetMethods() []reflect.Value {

	var injectorMethods []reflect.Value

	for i := range a.injectors {

		inj := a.injectors[i]
		method, err := validateDependencyInjector(inj)
		if err != nil {
			fmt.Println("❌", err)
			continue
		}

		injectorMethods = append(injectorMethods, method)
	}
	return injectorMethods
}

func validateDependencyInjector(inj any) (reflect.Value, error) {

	injType := reflect.TypeOf(inj)
	injValue := reflect.ValueOf(inj)

	method, ok := injType.MethodByName("Inject")

	if !ok {
		return reflect.Value{}, fmt.Errorf("%T n’a pas de méthode Inject", inj)
	}

	if method.Type.NumIn() != 2 {
		return reflect.Value{}, fmt.Errorf("%T.Inject doit avoir exactement 1 paramètre (string)", inj)
	}

	paramType := method.Type.In(1)
	if paramType.Kind() != reflect.String {
		return reflect.Value{}, fmt.Errorf("%T.Inject doit prendre un string (actuel : %v)", inj, paramType)
	}

	if method.Type.NumOut() != 1 {
		return reflect.Value{}, fmt.Errorf("%T.Inject doit retourner exactement une valeur", inj)
	}

	outType := method.Type.Out(0)
	if outType.Kind() != reflect.Ptr {
		return reflect.Value{}, fmt.Errorf("%T.Inject doit retourner un pointeur (actuel : %v)", inj, outType)
	}

	injectMethod := injValue.MethodByName("Inject")

	if !injectMethod.IsValid() {
		return reflect.Value{}, fmt.Errorf("%T.Inject introuvable sur la valeur", inj)
	}

	return injectMethod, nil
}
