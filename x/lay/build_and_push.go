package lay

import (
	"context"
	"errors"
	"fmt"
	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/x/mod"
	"go.uber.org/zap"
	"os"
	"path/filepath"
)

type BuildAndPushLayer struct {
	CBuilder *mod.ContainerBuilder
	Logger   *zap.Logger
}

type BuildAndPushLayerContext struct {
	ContextDir       string
	WorkDir          string
	SrcDockerfile    string
	ImageName        string
	Push             bool
	ExportToLocalTar bool
	LocalTarRef      string
	LocalTarPath     string
}

func (b *BuildAndPushLayer) Name() string {
	return "BuildAndPush"
}

func (b *BuildAndPushLayer) Run(ctx context.Context, pipeYardCtx pi.Ctx, c BuildAndPushLayerContext) error {

	if err := c.Validate(); err != nil {
		return err
	}

	if c.ContextDir == "./" || c.ContextDir == "" {
		c.ContextDir = pipeYardCtx.CurrentDir
	}
	if c.WorkDir == "./" || c.WorkDir == "" {
		c.WorkDir = pipeYardCtx.CurrentDir
	}

	b.Logger.Info("preparing docker build context",
		zap.String("contextDir", c.ContextDir),
		zap.String("workDir", c.WorkDir),
	)

	pipeyardDir := filepath.Join(c.ContextDir, ".pipeyard")

	if err := os.MkdirAll(pipeyardDir, 0o755); err != nil {
		return err
	}

	dstDockerfile := filepath.Join(
		pipeyardDir,
		fmt.Sprintf("Dockerfile.pipeyard.%d", os.Getpid()),
	)

	fp := mod.FileSystemProcessor{}

	if err := fp.CopyFile(c.SrcDockerfile, dstDockerfile); err != nil {
		return err
	}

	defer func() {
		_ = os.Remove(dstDockerfile)
		_ = os.Remove(pipeyardDir)
	}()

	containerSpecs := mod.ContainerSpec{
		WorkDir:        c.WorkDir,
		Logger:         b.Logger,
		DockerfilePath: dstDockerfile,
		ContextDir:     c.ContextDir,
	}

	b.Logger.Info("building image",
		zap.String("image", c.ImageName),
	)

	if c.Push {

		err := b.CBuilder.InitContainer(containerSpecs).Apply().Push(ctx, c.ImageName, true)

		if err != nil {
			return err
		}

		b.Logger.Info("Image successfully pushed",
			zap.String("image", c.ImageName),
		)

	}

	if c.ExportToLocalTar {

		o := mod.ContainerExportOptions{
			Mode:       mod.ContainerModeExportDocker,
			ImageRef:   c.LocalTarRef,
			OutputPath: c.LocalTarPath,
			Insecure:   true,
		}

		err := b.CBuilder.InitContainer(containerSpecs).Export(ctx, o)

		if err != nil {
			return err
		}

		b.Logger.Info("Image successfully exported to local",
			zap.String("image", c.ImageName),
		)

	}

	b.Logger.Info("Build and push completed")

	return nil
}

func (c BuildAndPushLayerContext) Validate() error {
	if c.ImageName == "" {
		return errors.New("image name is empty")
	}
	if c.SrcDockerfile == "" {
		return errors.New("src dockerfile is empty")
	}
	return nil
}
