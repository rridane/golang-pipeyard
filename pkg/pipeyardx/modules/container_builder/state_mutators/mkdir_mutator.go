package state_mutators

import (
	"github.com/moby/buildkit/client/llb"
	"os"
)

// MkdirMutator — crée un dossier
type MkdirMutator struct {
	Path string
	Mode os.FileMode
}

func (m *MkdirMutator) Mutate(st llb.State) llb.State {
	return st.File(llb.Mkdir(m.Path, m.Mode, llb.WithParents(true)))
}
