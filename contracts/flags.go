package contracts

type FlagSpec struct {
	Name        string
	Type        FlagType
	Default     any
	Description string
	Required    bool
}

type FlagType string
