package main

import (
	"context"
	"dagger.io/dagger"
	"fmt"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	kubernetes "k8s.io/client-go/kubernetes"
)

type BuildStage struct {
}

func (s *BuildStage) Configure(cmd *cobra.Command) {
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

func (s *BuildStage) Execute(ctx context.Context, cmd *cobra.Command, client *dagger.Client, logger *zap.Logger, tracer *trace.Tracer, k8sClient *kubernetes.Clientset) error {
	args := cmd.Flags().Args()
	application := args[0]
	version := args[1]

	force, _ := cmd.Flags().GetBool("force")

	logger.Info(fmt.Sprintf("🚀 Déploiement de %s version %s\n", application, version))
	logger.Info(fmt.Sprintf("⚡ Force: %t\n", force))

	return nil
}
