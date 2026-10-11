package pi

import (
	"slices"

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

// What a stage exposes is a contract, so a test has to be able to read it
// back: that a flag is still named timeout, still a duration, still required.

func (s *StageSpec) Command() string { return s.command }

func (s *StageSpec) Short() string { return s.short }

// Flags returns a copy, so reading the spec cannot change it.
func (s *StageSpec) Flags() []ru.Flag { return slices.Clone(s.args) }

// Flag finds one by name.
func (s *StageSpec) Flag(name string) (ru.Flag, bool) {
	for _, flag := range s.args {
		if flag.Name == name {
			return flag, true
		}
	}
	return ru.Flag{}, false
}
