package containers

import (
	"context"
	"errors"
	"fmt"
	"github.com/rridane/golang-pipeyard/helpers"
	"github.com/rridane/golang-pipeyard/pkg/container_builder"
	"github.com/rridane/golang-pipeyard/pkg/pipeyard_ctx"
	"go.uber.org/zap"
	"os"
	"path/filepath"
)

type BuildAndPushLayer struct {
	CBuilder *container_builder.ContainerBuilder
	Logger   *zap.Logger
}

type BuildAndPushLayerContext struct {
	ContextDir    string
	WorkDir       string
	SrcDockerfile string
	ImageName     string
}

func (b *BuildAndPushLayer) Name() string {
	return "BuildAndPush"
}

func (b *BuildAndPushLayer) Run(ctx context.Context, pipeYardCtx pipeyard_ctx.PipeyardCtx, c BuildAndPushLayerContext) error {

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

	if err := helpers.CopyFile(c.SrcDockerfile, dstDockerfile); err != nil {
		return err
	}

	defer func() {
		_ = os.Remove(dstDockerfile)
		_ = os.Remove(pipeyardDir)
	}()

	containerSpecs := container_builder.ContainerSpec{
		WorkDir:        c.WorkDir,
		Logger:         b.Logger,
		DockerfilePath: dstDockerfile,
		ContextDir:     c.ContextDir,
	}

	b.Logger.Info("building image",
		zap.String("image", c.ImageName),
	)

	err := b.CBuilder.InitContainer(containerSpecs).Apply().Push(ctx, c.ImageName, true)

	if err != nil {
		return err
	}

	b.Logger.Info("Image successfully pushed, buildAndPush layer completed",
		zap.String("image", c.ImageName),
	)

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
