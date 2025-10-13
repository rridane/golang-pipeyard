package injectors

import (
	"fmt"
	"helm.sh/helm/v3/pkg/action"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type HelmInjector struct {
}

func (h *HelmInjector) Inject(stageName string) *action.Configuration {

	actionConfig := new(action.Configuration)

	kubeConfig := getKubeConfig()
	if err := actionConfig.Init(kubeConfig, "default", "secret", func(format string, v ...interface{}) {
	}); err != nil {
		fmt.Printf("⚠️ Impossible d'initialiser Helm: %v\n", err)
		return nil
	}

	return actionConfig
}

func getKubeConfig() *genericclioptions.ConfigFlags {
	return genericclioptions.NewConfigFlags(false)
}
