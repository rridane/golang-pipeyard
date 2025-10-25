package state_mutators

import (
	"github.com/moby/buildkit/client/llb"
)

// RmMutator — supprime un fichier ou répertoire
type RmMutator struct {
	Path string
}

func (m *RmMutator) Mutate(st llb.State) llb.State {
	return st.File(llb.Rm(m.Path))
}
