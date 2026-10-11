package container_builder

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
)

const EndpointEnvVar = "BUILDKIT_HOST"

const defaultEndpoint = "tcp://127.0.0.1:1234"

// ContainerBuilder runs build recipes on a BuildKit daemon.
type ContainerBuilder struct {
	Endpoint string // empty → $BUILDKIT_HOST, then tcp://127.0.0.1:1234
	Logger   *zap.Logger
}

type BuildRequest struct {
	Dockerfile Dockerfile
	ContextDir string            // empty → current directory
	Target     string            // stage to reach; empty → the last one
	BuildArgs  map[string]string // ARG values
	Secrets    map[string]string // id → value, for RUN --mount=type=secret
	Platforms  []string          // ex. linux/amd64, linux/arm64
	Labels     map[string]string
}

func (c *ContainerBuilder) Build(ctx context.Context, req BuildRequest) (*Artifact, error) {
	daemon, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer daemon.Close()

	image, err := daemon.Solve(ctx, req, exportTarget{})
	if err != nil {
		return nil, fmt.Errorf("build %s: %w", req.describe(), err)
	}

	return &Artifact{builder: c, request: req, image: image}, nil
}

func (c *ContainerBuilder) endpoint() string {
	if c.Endpoint != "" {
		return c.Endpoint
	}
	if fromEnv := os.Getenv(EndpointEnvVar); fromEnv != "" {
		return fromEnv
	}
	return defaultEndpoint
}

func (c *ContainerBuilder) logger() *zap.Logger {
	if c.Logger != nil {
		return c.Logger
	}
	return zap.NewNop()
}

func (r BuildRequest) describe() string {
	if r.Target != "" {
		return fmt.Sprintf("(target %q)", r.Target)
	}
	return "(final image)"
}

func (r BuildRequest) contextDir() string {
	if r.ContextDir != "" {
		return r.ContextDir
	}
	return "."
}
