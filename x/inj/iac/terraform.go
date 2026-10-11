package iac

import (
	"fmt"
	"github.com/hashicorp/terraform-exec/tfexec"
	"os/exec"
)

type TerraformInjector struct {
	BaseDir string
}

func (t *TerraformInjector) Inject(stageName string) *tfexec.Terraform {
	tfPath, err := exec.LookPath("terraform")
	if err != nil {
		fmt.Printf("terraform not found on PATH: %v\n", err)
		return nil
	}

	tf, err := tfexec.NewTerraform(t.BaseDir, tfPath)
	if err != nil {
		fmt.Printf("terraform not initialised: %v\n", err)
		return nil
	}

	return tf
}
