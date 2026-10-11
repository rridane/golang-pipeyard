package container

import (
	"github.com/rridane/golang-pipeyard/x/mod"
)

type ContainerBuilderInjector struct {
	// Endpoint is the BuildKit daemon; empty falls back to $BUILDKIT_HOST and
	// then to tcp://127.0.0.1:1234.
	Endpoint string

	// Registries authenticates the registries a stage may push to, keyed by
	// host: "ghcr.io", "docker.io". Wiring them here means no stage has to
	// carry a credential through its flags. Leave it empty and the machine's
	// own docker config is read, which is what a laptop has after a
	// docker login.
	Registries map[string]mod.RegistryCredential
}

func (d *ContainerBuilderInjector) Inject(stageName string) *mod.ContainerBuilder {
	return &mod.ContainerBuilder{
		Endpoint:   d.Endpoint,
		Registries: d.Registries,
	}
}
