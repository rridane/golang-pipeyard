package ru

type Flag struct {
	Name        string
	Type        FlagType
	Default     any
	Description string
	Required    bool
}

type FlagType string

// Every FlagType here is declared and parsed by the runtime.
const (
	BoolFlag     FlagType = "bool"
	StringFlag   FlagType = "string"
	IntFlag      FlagType = "int"
	DurationFlag FlagType = "duration"
)
