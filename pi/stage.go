package pi

import (
	"github.com/rridane/golang-pipeyard/ru"
)

// Stage is one command of the pipeline.
//
// Every stage must also have an Execute method, which the interface
// deliberately does not declare:
//
//	Execute(ctx context.Context, args ru.ParsedArgs, deps ...) error
//
// Leaving it out is what lets a stage ask for exactly the dependencies it
// needs, each with its own type. The executor checks the signature at
// registration, since the compiler cannot.
type Stage interface {
	Configure() *StageSpec
}

type StageSpec struct {
	command string
	short   string
	long    string
	args    []ru.Flag
}

func NewStageSpec(name string) *StageSpec {
	return &StageSpec{
		command: name,
		short:   "",
		long:    "",
		args:    nil,
	}
}

func (s *StageSpec) AddFlag(name string, flagType ru.FlagType, description string, defaultValue any, required bool) {

	s.args = append(s.args, ru.Flag{
		Name:        name,
		Type:        flagType,
		Default:     defaultValue,
		Description: description,
		Required:    required,
	})

}

func (s *StageSpec) SetDescription(short string) {
	s.short = short
}
