package container_builder

import "github.com/moby/buildkit/client/llb"

type MutatorsInterface interface {
	Mutate(state llb.State) llb.State
}

func mutateState(state llb.State, mutators []MutatorsInterface) llb.State {

	for _, mutator := range mutators {
		state = mutator.Mutate(state)
	}

	return state

}
