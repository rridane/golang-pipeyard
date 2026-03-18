package pi

type Flag struct {
	Name        string
	Type        FlagType
	Default     any
	Description string
	Required    bool
}

type FlagType string
