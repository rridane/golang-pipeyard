package state_mutators

import (
	"github.com/moby/buildkit/client/llb"
	"os"
)

// MkfileMutator — crée un fichier avec un contenu
type MkfileMutator struct {
	Path string
	Mode os.FileMode
	Data []byte
}

func (m *MkfileMutator) Mutate(st llb.State) llb.State {
	return st.File(llb.Mkfile(m.Path, m.Mode, m.Data))
}
