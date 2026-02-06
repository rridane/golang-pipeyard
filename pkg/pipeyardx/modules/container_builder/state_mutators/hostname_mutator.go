package state_mutators

import "github.com/moby/buildkit/client/llb"

// HostnameMutator — définit le hostname du conteneur
type HostnameMutator struct {
	Name string
}

func (m *HostnameMutator) Mutate(st llb.State) llb.State {
	return st.Hostname(m.Name)
}
