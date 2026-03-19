package pi

import "github.com/rridane/golang-pipeyard/ru"

type Stage interface {
	Configure() StageSpec
	//Execute(ctx context.context, args parsedArgs, ...any) error is REQUIRED FOR ANY IMPLEMENTATION, WE DO NOT DECLARE IT FOR FLEXIBILITY
}

type StageSpec struct {
	command string
	short   string
	long    string
	args    []ru.Flag
}

func NewStageSpec(name string) StageSpec {
	return StageSpec{
		command: name,
		short:   "",
		long:    "",
		args:    nil,
	}
}

func (s StageSpec) AddFlag(name string, flagType ru.FlagType, description string, defaultValue string, required bool) {

	s.args = append(s.args, ru.Flag{
		Name:        name,
		Type:        flagType,
		Default:     defaultValue,
		Description: description,
		Required:    required,
	})

}

func (s StageSpec) SetDescription(long string) {
	s.long = long
}

//func (s StageSpec) SetShort(short string) {
//	s.short = short
//}
