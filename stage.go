package pipeline_provider

import (
	"github.com/spf13/cobra"
	"reflect"
)

type CommandStage interface {
	Configure(cmd *cobra.Command)
	//Execute(ctx context.context, cmd *cobra.Command, ...any) error is REQUIRED FOR ANY IMPLEMENTATION, WE DO NOT DECLARE IT FOR FLEXIBILITY
}

func getExecuteMethodType(stage CommandStage) (reflect.Method, bool) {
	stageVal := reflect.TypeOf(stage)
	m, _ := stageVal.MethodByName("Execute")
	return m, true
}

func callExecuteCommand(stage CommandStage, args []reflect.Value) {
	v := reflect.ValueOf(stage)
	meth := v.MethodByName("Execute")
	meth.Call(args)
}

func getConfigureMethod(stage CommandStage) (reflect.Method, bool) {
	stageVal := reflect.TypeOf(stage)
	m, _ := stageVal.MethodByName("Configure")
	return m, true
}
