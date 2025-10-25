package state_mutators

import (
	"github.com/moby/buildkit/client/llb"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

// PlatformMutator — définit la plateforme cible (GOOS/GOARCH)
type PlatformMutator struct {
	Platform ocispecs.Platform
}

func (m *PlatformMutator) Mutate(st llb.State) llb.State {
	return st.Platform(m.Platform)
}
