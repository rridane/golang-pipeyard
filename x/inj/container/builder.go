package container

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/rridane/golang-pipeyard/x/mod"
	"os"
	"path/filepath"
)

type ContainerBuilderInjector struct {
	DockerUsername                  string
	DockerPassword                  string
	CreateAndOverrideConfigJsonFile bool
}

func (d *ContainerBuilderInjector) Inject(stageName string) *mod.ContainerBuilder {

	if !d.CreateAndOverrideConfigJsonFile && (d.DockerUsername != "" || d.DockerPassword != "") {

		panic("Providing docker credentials without asking for config json file is invalid, set CreateAndOverrideConfigJsonFile to true or remove credentials")

	}

	if d.CreateAndOverrideConfigJsonFile {

		if err := d.generateDockerConfig(); err != nil {
			fmt.Println(err)
		}

	}

	return &mod.ContainerBuilder{}
}

// generateDockerConfig crée ou remplace ~/.docker/config.json
func (d *ContainerBuilderInjector) generateDockerConfig() error {
	// Fallback : si les credentials ne sont pas fournis, on tente via env
	if d.DockerUsername == "" {
		d.DockerUsername = os.Getenv("DOCKER_USERNAME")
	}
	if d.DockerPassword == "" {
		d.DockerPassword = os.Getenv("DOCKER_PASSWORD")
	}

	if d.DockerUsername == "" || d.DockerPassword == "" {
		return fmt.Errorf("missing Docker credentials (provide via flags or env vars)")
	}

	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", d.DockerUsername, d.DockerPassword)))

	// Structure JSON conforme
	config := map[string]interface{}{
		"auths": map[string]interface{}{
			"https://index.docker.io/v1/": map[string]string{
				"auth": auth,
			},
		},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal docker config: %w", err)
	}

	// Chemin ~/.docker/config.json
	dockerConfigDir := filepath.Join(os.Getenv("HOME"), ".docker")
	dockerConfigFile := filepath.Join(dockerConfigDir, "config.json")

	// Création du dossier si nécessaire
	if err := os.MkdirAll(dockerConfigDir, 0700); err != nil {
		return fmt.Errorf("failed to create .docker dir: %w", err)
	}

	// Écriture (avec écrasement)
	if err := os.WriteFile(dockerConfigFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write docker config: %w", err)
	}

	return nil
}
