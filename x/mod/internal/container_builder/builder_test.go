package container_builder

import "testing"

// Docker Hub is the one registry whose credentials are not stored under its
// host, and an image reference names it by leaving the host out.
func TestRegistryKey(t *testing.T) {
	const dockerHub = "https://index.docker.io/v1/"

	for host, want := range map[string]string{
		"":                          dockerHub,
		"docker.io":                 dockerHub,
		"index.docker.io":           dockerHub,
		"registry-1.docker.io":      dockerHub,
		"ghcr.io":                   "ghcr.io",
		"registry.example.com:5000": "registry.example.com:5000",
		"localhost:5555":            "localhost:5555",
	} {
		if got := registryKey(host); got != want {
			t.Errorf("registryKey(%q) = %q, want %q", host, got, want)
		}
	}
}

// Given no credentials the builder reads the machine's own config, which is
// what a laptop has after docker login.
func TestDockerConfig_FallsBackToTheMachine(t *testing.T) {
	withCredentials := (&daemon{registries: map[string]RegistryCredential{
		"ghcr.io": {Username: "someone", Password: "secret"},
	}}).dockerConfig()

	if _, ok := withCredentials.AuthConfigs["ghcr.io"]; !ok {
		t.Fatalf("want a credential under ghcr.io, got %v", withCredentials.AuthConfigs)
	}

	if (&daemon{}).dockerConfig() == nil {
		t.Fatal("an unconfigured builder must still produce a config")
	}
}
