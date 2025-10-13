package injectors

import (
	"context"
	"dagger.io/dagger"
	"fmt"
)

type DaggerInjector struct {
}

func (d *DaggerInjector) Inject(stageName string) *dagger.Client {
	ctx := context.Background()

	client, err := dagger.Connect(ctx)
	if err != nil {
		fmt.Printf("Impossible de connecter Dagger pour %s: %v\n", stageName, err)
		return nil
	}
	return client
}
