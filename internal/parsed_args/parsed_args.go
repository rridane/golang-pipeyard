package parsed_args

import (
	"fmt"
	"github.com/rridane/golang-pipeyard/x"
)

var _ x.ParsedArgsInterface = (*ParsedArgs)(nil)

type ParsedArgs struct {
	boolParams   map[string]bool
	stringParams map[string]string
	intParams    map[string]int
}

/* ----------------- String ----------------- */

func (a *ParsedArgs) StringOk(name string) (string, bool) {
	v, ok := a.stringParams[name]
	return v, ok
}

func (a *ParsedArgs) String(name string) string {
	v, _ := a.StringOk(name)
	return v
}

func (a *ParsedArgs) RequireString(name string) (string, error) {
	if v, ok := a.StringOk(name); ok {
		return v, nil
	}
	return "", fmt.Errorf("missing required string arg: %s", name)
}

/* ----------------- Int ----------------- */

func (a *ParsedArgs) IntOk(name string) (int, bool) {
	v, ok := a.intParams[name]
	return v, ok
}

func (a *ParsedArgs) Int(name string) int {
	v, _ := a.IntOk(name)
	return v
}

func (a *ParsedArgs) RequireInt(name string) (int, error) {
	if v, ok := a.IntOk(name); ok {
		return v, nil
	}
	return 0, fmt.Errorf("missing required int arg: %s", name)
}

/* ----------------- Bool ----------------- */

func (a *ParsedArgs) BoolOk(name string) (bool, bool) {
	v, ok := a.boolParams[name]
	return v, ok
}

func (a *ParsedArgs) Bool(name string) bool {
	v, _ := a.BoolOk(name)
	return v
}

func (a *ParsedArgs) RequireBool(name string) (bool, error) {
	if v, ok := a.BoolOk(name); ok {
		return v, nil
	}
	return false, fmt.Errorf("missing required bool arg: %s", name)
}

/* ----------------- Keys ----------------- */

func (a *ParsedArgs) Keys() (stringsKeys, boolKeys, intKeys []string) {
	for k := range a.stringParams {
		stringsKeys = append(stringsKeys, k)
	}
	for k := range a.boolParams {
		boolKeys = append(boolKeys, k)
	}
	for k := range a.intParams {
		intKeys = append(intKeys, k)
	}
	return
}

/* ----------------- Setters ----------------- */

// SetString enregistre (ou écrase) une valeur string.
func (a *ParsedArgs) SetString(name, value string) {
	if a.stringParams == nil {
		a.stringParams = make(map[string]string)
	}
	a.stringParams[name] = value
}

// SetBool enregistre (ou écrase) une valeur bool.
func (a *ParsedArgs) SetBool(name string, value bool) {
	if a.boolParams == nil {
		a.boolParams = make(map[string]bool)
	}
	a.boolParams[name] = value
}

// SetInt enregistre (ou écrase) une valeur int.
func (a *ParsedArgs) SetInt(name string, value int) {
	if a.intParams == nil {
		a.intParams = make(map[string]int)
	}
	a.intParams[name] = value
}
