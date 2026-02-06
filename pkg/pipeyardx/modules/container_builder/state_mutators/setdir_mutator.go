package state_mutators

import "github.com/moby/buildkit/client/llb"

type SetDirMutator struct {
	Dir string
}

func (e *SetDirMutator) Mutate(st llb.State) llb.State {

	if e.Dir != "" {
		return st.Dir(e.Dir)
	}
	return st

}
