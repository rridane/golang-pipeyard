package stages

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"strings"
)

type DockerBuildStage struct {
	ContextDir       string
	Dockerfile       string
	Destination      string
	BuildArgs        []string
	Target           string
	Cache            bool
	InsecureRegistry bool
	RegistryMirror   string
	Labels           map[string]string
	Push             bool
}

func (s *DockerBuildStage) Configure(cmd *cobra.Command) {
	cmd.Use = "docker-build"
	cmd.Short = "Construit une image Docker via Kaniko"

	cmd.Flags().StringVar(&s.ContextDir, "context", ".", "Chemin du contexte de build")
	cmd.Flags().StringVar(&s.Dockerfile, "dockerfile", "Dockerfile", "Chemin du Dockerfile")
	cmd.Flags().StringVar(&s.Destination, "destination", "", "Destination de l’image (registry/path:tag)")
	cmd.Flags().StringArrayVar(&s.BuildArgs, "build-arg", nil, "Arguments de build")
	cmd.Flags().StringVar(&s.Target, "target", "", "Étape Dockerfile cible (optionnelle)")
	cmd.Flags().BoolVar(&s.Cache, "cache", true, "Active le cache Kaniko")
	cmd.Flags().BoolVar(&s.InsecureRegistry, "insecure", false, "Autorise un registre non sécurisé")
	cmd.Flags().StringVar(&s.RegistryMirror, "mirror", "", "Miroir du registre (optionnel)")
	cmd.Flags().BoolVar(&s.Push, "push", true, "Pousse l’image après build")
}

func (s *DockerBuildStage) Execute(ctx context.Context, cmd *cobra.Command, logger *zap.Logger) error {
	logger.Info("🚀 Stage Kaniko build démarré")

	if s.Destination == "" {
		return fmt.Errorf("aucune destination fournie (--destination)")
	}

	args := []string{
		"--context=" + s.ContextDir,
		"--dockerfile=" + s.Dockerfile,
		"--destination=" + s.Destination,
	}

	if s.Target != "" {
		args = append(args, "--target="+s.Target)
	}
	if s.Cache {
		args = append(args, "--cache=true")
	}
	if s.InsecureRegistry {
		args = append(args, "--insecure")
	}
	if s.RegistryMirror != "" {
		args = append(args, "--registry-mirror="+s.RegistryMirror)
	}
	if !s.Push {
		args = append(args, "--no-push")
	}
	for _, ba := range s.BuildArgs {
		args = append(args, "--build-arg="+ba)
	}
	for k, v := range s.Labels {
		args = append(args, fmt.Sprintf("--label=%s=%s", k, v))
	}

	logger.Info("🔹 Exécution Kaniko", zap.Strings("args", args))

	cmdLine := append([]string{
		"docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/workspace", s.ContextDir),
		"gcr.io/kaniko-project/executor:latest",
	}, args...)

	logger.Info("🧱 Commande complète", zap.String("cmd", strings.Join(cmdLine, " ")))

	run := exec.CommandContext(ctx, cmdLine[0], cmdLine[1:]...)
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr

	if err := run.Run(); err != nil {
		return fmt.Errorf("échec du build kaniko : %w", err)
	}

	logger.Info("✅ Build Kaniko terminé avec succès", zap.String("image", s.Destination))
	return nil
}
