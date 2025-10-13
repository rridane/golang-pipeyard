package injectors

import (
	"fmt"
	"github.com/hashicorp/terraform-exec/tfexec"
	"os"
	"os/exec"
	"path/filepath"
)

type TerraformInjector struct {
	baseDir string
}

func (t *TerraformInjector) Inject(stageName string) *tfexec.Terraform {
	tfPath, err := findTerraformBinary()
	if err != nil {
		fmt.Printf("⚠️ Terraform non trouvé: %v\n", err)
		return nil
	}

	workingDir := t.getWorkingDir(stageName)
	if workingDir == "" {
		return nil
	}

	tf, err := tfexec.NewTerraform(workingDir, tfPath)
	if err != nil {
		fmt.Printf("⚠️ Impossible de créer Terraform: %v\n", err)
		return nil
	}

	t.setupBackend(tf, stageName)

	return tf
}

func (t *TerraformInjector) getWorkingDir(stageName string) string {

	possibleDirs := []string{
		filepath.Join(t.baseDir, stageName),
		filepath.Join(t.baseDir, "terraform", stageName),
		filepath.Join(".", "infra", stageName),
		filepath.Join(".", "terraform", stageName),
	}

	for _, dir := range possibleDirs {
		if _, err := os.Stat(filepath.Join(dir, "main.tf")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, "*.tf")); err == nil {
			return dir
		}
	}

	fmt.Printf("⚠️ Aucun répertoire Terraform trouvé pour %s\n", stageName)
	return ""
}

func (t *TerraformInjector) setupBackend(tf *tfexec.Terraform, stageName string) {
	if os.Getenv("TF_BACKEND_S3") != "" {
	}
}

func findTerraformBinary() (string, error) {
	if path, err := exec.LookPath("terraform"); err == nil {
		return path, nil
	}

	commonPaths := []string{
		"/usr/local/bin/terraform",
		"/opt/homebrew/bin/terraform",
		"./bin/terraform",
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("terraform binary not found")
}
