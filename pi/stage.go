package pi

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
