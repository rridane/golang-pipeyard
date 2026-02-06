package dependency_injection

import (
	"fmt"
	"reflect"
)

type DependencyInjector[T any] interface {
	Inject(stageName string) *T
}

// return serviceA reflect value for a method Method(serviceA) if injector of ServiceA is active
// Method ...params
func GetDependencies(m reflect.Method, injectMethods []reflect.Value, stageName string) ([]reflect.Value, error) {

	//start := time.Now()

	var arg []reflect.Value
	var missingDeps []string
	numParams := m.Type.NumIn()

	// 1: context.Context
	// 2: *parsedArgs
	// 3+: dépendances
	for p := 3; p < numParams; p++ {

		indexPParamType := m.Type.In(p)
		found := false

		for i := 0; i < len(injectMethods); i++ {

			//start_method := time.Now()

			injectorMethod := injectMethods[i]
			returnType := injectorMethod.Type().Out(0)

			if !returnType.AssignableTo(indexPParamType) {
				continue
			}

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

			//elapsed_method := time.Since(start_method)
			//fmt.Printf("Traitement de %s: %s\n", injectorMethod, elapsed_method)
		}

		if !found {
			missingDeps = append(missingDeps, indexPParamType.String())
		}

		//elapsed := time.Since(start)
		//fmt.Printf("⏱️ Durée de fetch des dépendances: %s\n", elapsed)

	}

	if len(missingDeps) > 0 {
		return nil, fmt.Errorf("dépendances manquantes: %v", missingDeps)
	}

	return arg, nil
}

func ValidateDependencyInjector(inj any) (reflect.Value, error) {

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

	//outType := method.Type.Out(0)
	//if outType.Kind() != reflect.Ptr {
	//	return reflect.Value{}, fmt.Errorf("%T.Inject doit retourner un pointeur (actuel : %v)", inj, outType)
	//}

	injectMethod := injValue.MethodByName("Inject")

	if !injectMethod.IsValid() {
		return reflect.Value{}, fmt.Errorf("%T.Inject introuvable sur la valeur", inj)
	}

	return injectMethod, nil
}
