package container_builder

import (
	"context"
	"fmt"
	"github.com/docker/cli/cli/config"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/client/llb/imagemetaresolver"
	"github.com/moby/buildkit/frontend/dockerfile/dockerfile2llb"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/rridane/golang-pipeyard/modules/container_builder/state_mutators"
	"github.com/tonistiigi/fsutil"
	"go.uber.org/zap"
	"os"
)

const HostCurrentDirFs = "host"

type ContainerBuilder struct {
	specs ContainerSpec
}

type ContainerSpec struct {
	Image string
	//Mounts  map[string]string // logicalID -> containerPath
	Env     map[string]string
	WorkDir string
	state   llb.State
	// EntryPoint par défaut pour les exec - sh, bash, fish, default sh...
	Logger         *zap.Logger
	EntryPoint     string
	DockerfilePath string
	ContextDir     string
}

func (c *ContainerBuilder) InitContainer(specs ContainerSpec) *ContainerBuilder {

	c.specs = specs

	if c.specs.EntryPoint == "" {
		c.specs.EntryPoint = "sh"
	}

	switch {
	case c.specs.DockerfilePath != "":
		return c.initFromDockerfile()
	case c.specs.Image != "":
		c.specs.state = llb.Image(c.specs.Image, llb.WithMetaResolver(imagemetaresolver.Default()))
		return c
	default:
		panic("neither Dockerfile nor Image specified")
	}

	return c

}

func (c *ContainerBuilder) initFromDockerfile() *ContainerBuilder {
	ctx := context.Background()

	// Lis le contenu du Dockerfile
	dt, err := os.ReadFile(c.specs.DockerfilePath)

	if err != nil {
		c.specs.Logger.Error("cannot read dockerfile", zap.Error(err))
		return nil
	}

	// Convertit le Dockerfile → graphe LLB
	st, _, _, _, err := dockerfile2llb.Dockerfile2LLB(ctx, dt, dockerfile2llb.ConvertOpt{
		MetaResolver: imagemetaresolver.Default(),
	})
	if err != nil {
		c.specs.Logger.Error("dockerfile2llb failed", zap.Error(err))
		return nil
	}

	c.specs.state = *st
	return c
}

func (c *ContainerBuilder) Exec(cmd string) *ContainerBuilder {

	var mountOptions []llb.RunOption

	//for id, containerPath := range c.specs.Mounts {
	//	mountOptions = append(mountOptions, llb.AddMount(containerPath, llb.Local(id)))
	//}

	mutators := []MutatorsInterface{
		&state_mutators.EnvMutators{Env: c.specs.Env},
		&state_mutators.SetDirMutator{Dir: c.specs.WorkDir},
		&state_mutators.ExecMutator{Options: append([]llb.RunOption{llb.Args([]string{c.specs.EntryPoint, "-c", cmd})}, mountOptions...)},
	}

	c.processMutators(mutators)

	return c
}

func (c *ContainerBuilder) CopyFromHost(src string, dst string) *ContainerBuilder {

	localState := llb.Local(HostCurrentDirFs)
	if _, err := os.Stat("blabla"); err != nil {
		c.specs.Logger.Error(fmt.Sprintf("❌ Le fichier 'blabla' n'existe pas sur le host"))
	}
	mutators := []MutatorsInterface{
		&state_mutators.CopyMutator{
			Source: localState,
			Src:    src,
			Dst:    dst,
		},
	}

	c.processMutators(mutators)

	return c

}

func (c *ContainerBuilder) CopyToHost(src, dstDir string) *ContainerBuilder {

	ctx := context.Background()

	outputState := llb.Scratch().File(llb.Copy(c.specs.state, src, "/"))

	def, err := outputState.Marshal(ctx)
	if err != nil {
		c.specs.Logger.Error("marshal failed in CopyToHost", zap.Error(err))
		return c
	}

	cli, err := client.New(ctx, "tcp://0.0.0.0:1234")
	if err != nil {
		c.specs.Logger.Error("buildkit connect failed", zap.Error(err))
		return c
	}
	defer cli.Close()

	statusCh := make(chan *client.SolveStatus)
	go c.streamStatus(statusCh)

	fs, err := fsutil.NewFS(".")
	if err != nil {
		c.specs.Logger.Error("fsutil.NewFS failed", zap.Error(err))
		return c
	}

	// 📁 Crée le dossier de destination
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		c.specs.Logger.Error("failed to create output dir", zap.Error(err))
		return c
	}

	_, err = cli.Solve(ctx, def, client.SolveOpt{
		LocalMounts: map[string]fsutil.FS{
			"host": fs, // ⚠️ doit correspondre à ton llb.Local("host")
		},
		Exports: []client.ExportEntry{
			{
				Type:      client.ExporterLocal,
				Attrs:     map[string]string{"dest": dstDir},
				OutputDir: dstDir,
			},
		},
	}, statusCh)

	if err != nil {
		c.specs.Logger.Error("CopyToHost failed", zap.Error(err))
		return c
	}

	c.specs.Logger.Info(fmt.Sprintf("✅ Exported snapshot to %s", dstDir))
	return c
}

func (c *ContainerBuilder) Push(ctx context.Context, imageRef string, insecure bool) error {
	cli, err := client.New(ctx, "tcp://0.0.0.0:1234")
	if err != nil {
		return fmt.Errorf("buildkit connect failed: %w", err)
	}
	defer cli.Close()

	def, err := c.specs.state.Marshal(ctx)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// === 1️⃣ Charger ou générer le fichier config Docker ===
	cfgFile := config.LoadDefaultConfigFile(os.Stderr)

	// === 2️⃣ Créer un provider d’authentification ===
	authProvider := authprovider.NewDockerAuthProvider(authprovider.DockerAuthProviderConfig{
		ConfigFile: cfgFile,
	})

	// === 3️⃣ Préparer les options de push ===
	exportAttrs := map[string]string{
		"name": imageRef,
		"push": "true",
	}
	if insecure {
		exportAttrs["registry.insecure"] = "true"
	}

	statusCh := make(chan *client.SolveStatus)
	go c.streamStatus(statusCh)

	fs, err := fsutil.NewFS(".")
	if err != nil {
		return fmt.Errorf("failed to access host fs: %w", err)
	}

	// === 4️⃣ Solve avec session attachée (auth provider) ===
	_, err = cli.Solve(ctx, def, client.SolveOpt{
		LocalMounts: map[string]fsutil.FS{
			HostCurrentDirFs: fs,
		},
		Session: []session.Attachable{
			authProvider, //
		},
		Exports: []client.ExportEntry{
			{
				Type:  "image",
				Attrs: exportAttrs,
			},
		},
	}, statusCh)

	if err != nil {
		c.specs.Logger.Error("❌ Push failed", zap.Error(err))
		return fmt.Errorf("push failed: %w", err)
	}

	c.specs.Logger.Info("✅ Image pushed successfully",
		zap.String("ref", imageRef),
		zap.String("registry", imageRef),
	)
	return nil
}

func (c *ContainerBuilder) processMutators(mutators []MutatorsInterface) *ContainerBuilder {

	c.specs.state = mutateState(c.specs.state, mutators)

	return c
}

func (c *ContainerBuilder) Apply() *ContainerBuilder {
	ctx := context.Background()

	def, err := c.specs.state.Marshal(ctx)

	if err != nil {
		c.specs.Logger.Error("marshal failed", zap.Error(err))
		return nil
	}

	err = c.solveGraph(ctx, def)

	if err != nil {
		return nil
	}

	return c
}

func (c *ContainerBuilder) solveGraph(ctx context.Context, def *llb.Definition) error {

	cli, err := client.New(ctx, "tcp://0.0.0.0:1234")
	if err != nil {
		c.specs.Logger.Error("buildkit connect failed", zap.Error(err))
		return nil
	}
	defer func(cli *client.Client) {
		err := cli.Close()
		if err != nil {
			c.specs.Logger.Error("close failed", zap.Error(err))
		}
	}(cli)

	statusCh := make(chan *client.SolveStatus)
	go c.streamStatus(statusCh)

	fs, err := fsutil.NewFS(".")
	_, err = cli.Solve(ctx, def, client.SolveOpt{
		LocalMounts: map[string]fsutil.FS{HostCurrentDirFs: fs},
	}, statusCh)

	if err != nil {
		c.specs.Logger.Error("solve failed", zap.Error(err))
	}

	return nil
}

func (c *ContainerBuilder) streamStatus(ch <-chan *client.SolveStatus) {
	for s := range ch {

		for _, v := range s.Vertexes {
			if v.Error != "" {
				c.specs.Logger.Error(fmt.Sprintf("❌ %s: %s", v.Name, v.Error))
			} else {
				if v.Completed == nil {
					c.specs.Logger.Info(fmt.Sprintf("🔹  %s", v.Name))
				}
			}
		}
		for _, l := range s.Logs {
			if l.Stream == 1 {
				c.specs.Logger.Info(fmt.Sprintf("%s", string(l.Data)))
			} else {
				c.specs.Logger.Info(fmt.Sprintf("%s", string(l.Data)))
			}
		}
	}
}
