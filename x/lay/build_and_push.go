package lay

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/x/mod"
	"go.uber.org/zap"
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
	Insecure         bool
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

	recette, err := os.ReadFile(c.SrcDockerfile)
	if err != nil {
		return fmt.Errorf("read dockerfile %s: %w", c.SrcDockerfile, err)
	}

	b.Logger.Info("building image",
		zap.String("image", c.ImageName),
		zap.String("contextDir", c.ContextDir),
		zap.String("dockerfile", c.SrcDockerfile),
	)

	artefact, err := b.CBuilder.Build(ctx, mod.BuildRequest{
		Dockerfile: mod.Dockerfile{Template: string(recette)},
		ContextDir: c.ContextDir,
	})
	if err != nil {
		return err
	}

	if c.Push {
		if err := b.push(ctx, artefact, c); err != nil {
			return err
		}
		b.Logger.Info("Image successfully pushed", zap.String("image", c.ImageName))
	}

	if c.ExportToLocalTar {
		if err := artefact.SaveTar(ctx, c.LocalTarRef, c.LocalTarPath); err != nil {
			return err
		}
		b.Logger.Info("Image successfully exported to local", zap.String("image", c.ImageName))
	}

	b.Logger.Info("Build and push completed")

	return nil
}

// push publishes the image, over plain HTTP when the registry has no TLS.
func (b *BuildAndPushLayer) push(ctx context.Context, artefact *mod.Artifact, c BuildAndPushLayerContext) error {
	if c.Insecure {
		return artefact.PushInsecure(ctx, c.ImageName)
	}
	return artefact.Push(ctx, c.ImageName)
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
