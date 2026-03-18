package state_mutators

import "github.com/moby/buildkit/client/llb"

// UserMutator — change l'utilisateur courant
type UserMutator struct {
	User string
}

func (m *UserMutator) Mutate(st llb.State) llb.State {
	if m.User != "" {
		return st.User(m.User)
	}
	return st
}
