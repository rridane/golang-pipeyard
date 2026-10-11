package container_builder

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
)

// EndpointEnvVar points at another BuildKit daemon without recompiling.
const EndpointEnvVar = "BUILDKIT_HOST"

const defaultEndpoint = "tcp://127.0.0.1:1234"

// ContainerBuilder runs build recipes on a BuildKit daemon.
//
// It composes nothing itself: the recipe is a Dockerfile and BuildKit is its
// interpreter. The builder's job is to carry the recipe to the daemon and to
// return a usable Artifact.
type ContainerBuilder struct {
	Endpoint string // empty → $BUILDKIT_HOST, then tcp://127.0.0.1:1234
	Logger   *zap.Logger
}

// BuildRequest describes a build: what to build, from which context, with
// which parameters and which secrets.
type BuildRequest struct {
	Dockerfile Dockerfile
	ContextDir string            // empty → current directory
	Target     string            // stage to reach; empty → the last one
	BuildArgs  map[string]string // ARG values
	Secrets    map[string]string // id → value, for RUN --mount=type=secret
	Platforms  []string          // ex. linux/amd64, linux/arm64
	Labels     map[string]string
}

// Build runs the recipe and returns the resulting artifact. A failed build
// returns an error: there is no path by which a failure could pass for a
// success.
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

// endpoint resolves the daemon address: field, then env var, then default.
func (c *ContainerBuilder) endpoint() string {
	if c.Endpoint != "" {
		return c.Endpoint
	}
	if fromEnv := os.Getenv(EndpointEnvVar); fromEnv != "" {
		return fromEnv
	}
	return defaultEndpoint
}

// logger always returns a usable logger, even when the caller provided none.
func (c *ContainerBuilder) logger() *zap.Logger {
	if c.Logger != nil {
		return c.Logger
	}
	return zap.NewNop()
}

// describe names the build in error messages.
func (r BuildRequest) describe() string {
	if r.Target != "" {
		return fmt.Sprintf("(target %q)", r.Target)
	}
	return "(final image)"
}

// contextDir returns the build context directory, the current one by default.
func (r BuildRequest) contextDir() string {
	if r.ContextDir != "" {
		return r.ContextDir
	}
	return "."
}
