package ru

import (
	"fmt"
	"time"
)

type ParsedArgs struct {
	boolParams     map[string]bool
	stringParams   map[string]string
	intParams      map[string]int
	durationParams map[string]time.Duration
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

/* ----------------- Duration ----------------- */

func (a *ParsedArgs) DurationOk(name string) (time.Duration, bool) {
	v, ok := a.durationParams[name]
	return v, ok
}

func (a *ParsedArgs) Duration(name string) time.Duration {
	v, _ := a.DurationOk(name)
	return v
}

func (a *ParsedArgs) RequireDuration(name string) (time.Duration, error) {
	if v, ok := a.DurationOk(name); ok {
		return v, nil
	}
	return 0, fmt.Errorf("missing required duration arg: %s", name)
}

/* ----------------- Keys ----------------- */

func (a *ParsedArgs) Keys() (stringsKeys, boolKeys, intKeys, durationKeys []string) {
	for k := range a.stringParams {
		stringsKeys = append(stringsKeys, k)
	}
	for k := range a.boolParams {
		boolKeys = append(boolKeys, k)
	}
	for k := range a.intParams {
		intKeys = append(intKeys, k)
	}
	for k := range a.durationParams {
		durationKeys = append(durationKeys, k)
	}
	return
}

/* ----------------- Setters ----------------- */

func (a *ParsedArgs) SetString(name, value string) {
	if a.stringParams == nil {
		a.stringParams = make(map[string]string)
	}
	a.stringParams[name] = value
}

func (a *ParsedArgs) SetBool(name string, value bool) {
	if a.boolParams == nil {
		a.boolParams = make(map[string]bool)
	}
	a.boolParams[name] = value
}

func (a *ParsedArgs) SetInt(name string, value int) {
	if a.intParams == nil {
		a.intParams = make(map[string]int)
	}
	a.intParams[name] = value
}

func (a *ParsedArgs) SetDuration(name string, value time.Duration) {
	if a.durationParams == nil {
		a.durationParams = make(map[string]time.Duration)
	}
	a.durationParams[name] = value
}
