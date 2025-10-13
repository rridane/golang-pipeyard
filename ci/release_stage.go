package main

import (
	"context"
	"dagger.io/dagger"
	"fmt"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ReleaseStage struct {
}

func (s *ReleaseStage) Configure(cmd *cobra.Command) {
	cmd.Use = "deploy <application> <version>"
	cmd.Short = "Déploie une application sur le cluster"
	cmd.Long = `Déploie une application sur le cluster Kubernetes.

Arguments:
  APPLICATION   Nom de l'application à déployer
  VERSION       Version de l'application`

	cmd.Example = `  kubeapp deploy myapp 1.0.0 -n production
  kubeapp deploy frontend latest --replicas=5 --force`

	cmd.Args = cobra.ExactArgs(2)

	cmd.Flags().Bool("force", false, "Forcer le déploiement")

	err := cmd.MarkFlagRequired("namespace")

	if err != nil {
		return
	}
}

func (s *ReleaseStage) Execute(ctx context.Context, cmd *cobra.Command, client *dagger.Client, logger *zap.Logger, tracer *trace.Tracer, k8sClient *kubernetes.Clientset) error {
	args := cmd.Flags().Args()
	application := args[0]
	version := args[1]

	namespace, _ := cmd.Flags().GetString("namespace")
	replicas, _ := cmd.Flags().GetInt("replicas")
	force, _ := cmd.Flags().GetBool("force")

	logger.Info(fmt.Sprintf("🚀 Déploiement de %s version %s\n", application, version))
	logger.Info(fmt.Sprintf("📁 Namespace: %s\n", namespace))
	logger.Info(fmt.Sprintf("🔢 Réplicas: %d\n", replicas))
	logger.Info(fmt.Sprintf("⚡ Force: %t\n", force))

	pods, err := k8sClient.CoreV1().Pods(namespace).List(ctx, v1.ListOptions{})

	if err != nil {
		return nil
	}

	fmt.Printf(pods.String())

	return nil
}
