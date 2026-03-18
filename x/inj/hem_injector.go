package inj

import (
	"fmt"
	"helm.sh/helm/v3/pkg/action"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// HelmInjector configure une instance Helm explicitement.
// Si aucun KubeConfigPath n'est fourni, il tentera d'utiliser la configuration in-cluster.
type HelmInjector struct {
	KubeConfigPath string // Chemin vers le fichier kubeconfig (facultatif)
	Namespace      string // Namespace cible (ex: "default", "prod")
	StorageDriver  string // "secret" ou "configmap" (Helm backend)
}

// Inject initialise et renvoie la configuration Helm pour le stage donné.
func (h *HelmInjector) Inject(stageName string) *action.Configuration {
	fmt.Printf("🔧 Injection de Helm Configuration pour le stage %s\n", stageName)

	if h.Namespace == "" {
		h.Namespace = "default"
	}
	if h.StorageDriver == "" {
		h.StorageDriver = "secret"
	}

	configFlags := genericclioptions.NewConfigFlags(false)
	if h.KubeConfigPath != "" {
		configFlags.KubeConfig = &h.KubeConfigPath
	}

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(configFlags, h.Namespace, h.StorageDriver, func(format string, v ...interface{}) {
		fmt.Printf(format+"\n", v...)
	}); err != nil {
		fmt.Printf("⚠️ Impossible d'initialiser Helm: %v\n", err)
		return nil
	}

	fmt.Printf("✅ Helm initialisé (namespace=%s, driver=%s)\n", h.Namespace, h.StorageDriver)
	return actionConfig
}
