package state_mutators

import (
	"github.com/moby/buildkit/client/llb"
)

// SymlinkMutator — crée un lien symbolique
type SymlinkMutator struct {
	OldPath string
	NewPath string
}

func (m *SymlinkMutator) Mutate(st llb.State) llb.State {
	return st.File(llb.Symlink(m.OldPath, m.NewPath))
}
