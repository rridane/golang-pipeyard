package inj

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

type K8sInjector struct{}

func (k *K8sInjector) Inject(stageName string) *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			fmt.Printf("Impossible de charger la config k8s: %v\n", err)
			return nil
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Impossible de créer le client k8s: %v\n", err)
		return nil
	}
	return clientset
}
