package ru

type Flag struct {
	Name        string
	Type        FlagType
	Default     any
	Description string
	Required    bool
}

type FlagType string

const (
	BoolFlag     FlagType = "bool"
	StringFlag   FlagType = "string"
	IntFlag      FlagType = "int"
	DurationFlag FlagType = "duration"
	FilePathFlag FlagType = "filepath"
	URLFlag      FlagType = "url"
)
