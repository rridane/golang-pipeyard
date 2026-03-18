package state_mutators

import (
	"github.com/moby/buildkit/client/llb"
)

type EnvMutators struct {
	Env map[string]string
}

func (e *EnvMutators) Mutate(st llb.State) llb.State {

	for k, v := range e.Env {
		st = st.AddEnv(k, v)
	}
	return st
}
