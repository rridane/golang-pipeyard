package ru

type Flag struct {
	Name        string
	Type        FlagType
	Default     any
	Description string
	Required    bool
}

type FlagType string

// Every FlagType listed here is declared and parsed by the runtime. A type that
// the runtime cannot handle has no reason to exist here: declaring it would make
// the stage impossible to run.
const (
	BoolFlag     FlagType = "bool"
	StringFlag   FlagType = "string"
	IntFlag      FlagType = "int"
	DurationFlag FlagType = "duration"
)
