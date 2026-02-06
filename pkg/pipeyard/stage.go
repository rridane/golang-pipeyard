package pipeyard

import (
	"reflect"
)

type Stage interface {
	Configure() StageSpec
	//Execute(ctx context.context, args parsedArgs, ...any) error is REQUIRED FOR ANY IMPLEMENTATION, WE DO NOT DECLARE IT FOR FLEXIBILITY
}

type StageSpec struct {
	Command string
	Short   string
	Long    string
	Args    []FlagSpec
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
