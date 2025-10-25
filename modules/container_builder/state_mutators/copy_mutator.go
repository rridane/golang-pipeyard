package state_mutators

import (
	"github.com/moby/buildkit/client/llb"
)

//// Sources typiques
// stImg   := llb.Image("alpine:3.20")      // image OCI/Docker
// stLocal := llb.Local("src")              // dossier local
// stGit   := llb.Git("[https://repo.git](https://repo.git)")   // checkout Git
// stHTTP  := llb.HTTP("[https://host/file](https://host/file)") // téléchargement HTTP
// stOCI   := llb.OCILayout("/path/to/oci") // layout OCI local

// CopyMutator — copie des fichiers d'une source vers une destination
type CopyMutator struct {
	Source llb.State
	Src    string
	Dst    string
}

func (m *CopyMutator) Mutate(st llb.State) llb.State {
	return st.File(llb.Copy(m.Source, m.Src, m.Dst))
}
