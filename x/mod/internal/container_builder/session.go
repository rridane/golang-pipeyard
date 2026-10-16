package container_builder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	"github.com/moby/buildkit/frontend/dockerfile/dockerfile2llb"
	"github.com/moby/buildkit/frontend/dockerui"
	gateway "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/session/secrets/secretsprovider"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	"github.com/moby/patternmatcher/ignorefile"
	"github.com/tonistiigi/fsutil"
	"go.uber.org/zap"
)

const frontendName = "pipeyard"

type daemon struct {
	client     *client.Client
	logger     *zap.Logger
	registries map[string]RegistryCredential
}

func (c *ContainerBuilder) connect(ctx context.Context) (*daemon, error) {
	address := c.endpoint()

	conn, err := client.New(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("connect to BuildKit at %s: %w", address, err)
	}

	return &daemon{client: conn, logger: c.logger(), registries: c.Registries}, nil
}

func (d *daemon) Close() {
	if err := d.client.Close(); err != nil {
		d.logger.Warn("fermeture de la conn BuildKit", zap.Error(err))
	}
}

func (d *daemon) Solve(ctx context.Context, req BuildRequest, target exportTarget) (*dockerspec.DockerOCIImage, error) {
	options, err := d.solveOptions(req, target)
	if err != nil {
		return nil, err
	}

	var image *dockerspec.DockerOCIImage

	_, err = d.client.Build(ctx, options, frontendName,
		func(ctx context.Context, gw gateway.Client) (*gateway.Result, error) {
			state, config, err := req.compile(ctx, gw)
			if err != nil {
				return nil, err
			}
			image = config

			ref, err := req.execute(ctx, gw, target.narrow(state))
			if err != nil {
				return nil, err
			}
			return req.deliver(ref, config)
		},
		d.watch(ctx))

	if err != nil {
		return nil, err
	}
	return image, nil
}

func (d *daemon) ReadFile(ctx context.Context, req BuildRequest, path string) ([]byte, error) {
	options, err := d.solveOptions(req, exportTarget{})
	if err != nil {
		return nil, err
	}

	var content []byte

	_, err = d.client.Build(ctx, options, frontendName,
		func(ctx context.Context, gw gateway.Client) (*gateway.Result, error) {
			state, config, err := req.compile(ctx, gw)
			if err != nil {
				return nil, err
			}

			ref, err := req.execute(ctx, gw, state)
			if err != nil {
				return nil, err
			}

			content, err = ref.ReadFile(ctx, gateway.ReadRequest{Filename: path})
			if err != nil {
				return nil, fmt.Errorf("read %s from image: %w", path, err)
			}
			return req.deliver(ref, config)
		},
		d.watch(ctx))

	if err != nil {
		return nil, err
	}
	return content, nil
}

func (d *daemon) watch(ctx context.Context) chan *client.SolveStatus {
	statusCh := make(chan *client.SolveStatus)
	go progress{logger: d.logger}.consume(statusCh)
	return statusCh
}

// compile turns the recipe into an LLB graph and returns the configuration of
// the image the Dockerfile describes (ENTRYPOINT, USER, ENV, ports, labels).
func (r BuildRequest) compile(ctx context.Context, gw gateway.Client) (llb.State, *dockerspec.DockerOCIImage, error) {
	buildCtx := llb.Local(dockerui.DefaultLocalNameContext)

	result, err := dockerfile2llb.Dockerfile2LLB(ctx, []byte(r.Dockerfile.Render()),
		dockerfile2llb.ConvertOpt{
			Config: dockerui.Config{
				BuildArgs: r.BuildArgs,
				Target:    r.Target,
				Labels:    r.Labels,
			},
			MetaResolver: gw,
			MainContext:  &buildCtx,
		})
	if err != nil {
		return llb.State{}, nil, fmt.Errorf("invalid dockerfile: %w", err)
	}
	return result.State, result.Image, nil
}

func (r BuildRequest) execute(ctx context.Context, gw gateway.Client, state llb.State) (gateway.Reference, error) {
	definition, err := state.Marshal(ctx)
	if err != nil {
		return nil, fmt.Errorf("marshal build graph: %w", err)
	}

	result, err := gw.Solve(ctx, gateway.SolveRequest{Definition: definition.ToPB()})
	if err != nil {
		return nil, err
	}

	ref, err := result.SingleRef()
	if err != nil {
		return nil, fmt.Errorf("unexpected build result: %w", err)
	}
	return ref, nil
}

// deliver attaches the image configuration to the result. Without this step
// the pushed image would have neither ENTRYPOINT nor USER: it would not start.
func (r BuildRequest) deliver(ref gateway.Reference, image *dockerspec.DockerOCIImage) (*gateway.Result, error) {
	config, err := json.Marshal(image)
	if err != nil {
		return nil, fmt.Errorf("marshal image config: %w", err)
	}

	result := gateway.NewResult()
	result.SetRef(ref)
	result.AddMeta(exptypes.ExporterImageConfigKey, config)
	return result, nil
}

func (d *daemon) solveOptions(req BuildRequest, target exportTarget) (client.SolveOpt, error) {
	buildCtx, err := req.buildContext()
	if err != nil {
		return client.SolveOpt{}, err
	}

	return client.SolveOpt{
		LocalMounts: map[string]fsutil.FS{dockerui.DefaultLocalNameContext: buildCtx},
		Session:     d.attachables(req),
		Exports:     target.entries,
	}, nil
}

// buildContext reads .dockerignore from the context directory and filters the
// tree accordingly. Without it every build ships .git and whatever else the
// directory happens to hold, and fails outright on a file it cannot read.
func (r BuildRequest) buildContext() (fsutil.FS, error) {
	dir := r.contextDir()

	tree, err := fsutil.NewFS(dir)
	if err != nil {
		return nil, fmt.Errorf("build context %s: %w", dir, err)
	}

	excluded, err := r.dockerignore(dir)
	if err != nil {
		return nil, err
	}
	if len(excluded) == 0 {
		return tree, nil
	}

	filtered, err := fsutil.NewFilterFS(tree, &fsutil.FilterOpt{ExcludePatterns: excluded})
	if err != nil {
		return nil, fmt.Errorf("apply %s: %w", dockerui.DefaultDockerignoreName, err)
	}
	return filtered, nil
}

func (r BuildRequest) dockerignore(dir string) ([]string, error) {
	path := filepath.Join(dir, dockerui.DefaultDockerignoreName)

	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	defer file.Close()

	patterns, err := ignorefile.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return patterns, nil
}

// attachables grants the build the registry credentials this builder was
// configured with and the secrets this request declares, for the duration of
// the build only.
func (d *daemon) attachables(req BuildRequest) []session.Attachable {
	auth := authprovider.NewDockerAuthProvider(authprovider.DockerAuthProviderConfig{
		AuthConfigProvider: authprovider.LoadAuthConfig(d.dockerConfig()),
	})

	attachables := []session.Attachable{auth}

	if len(req.Secrets) > 0 {
		values := make(map[string][]byte, len(req.Secrets))
		for id, secret := range req.Secrets {
			values[id] = []byte(secret)
		}
		attachables = append(attachables, secretsprovider.FromMap(values))
	}
	return attachables
}

// dockerConfig carries the credentials the builder was given, or the machine's
// own when it was given none. It is held in memory: nothing is written to
// ~/.docker/config.json, and nothing already there is overwritten.
func (d *daemon) dockerConfig() *configfile.ConfigFile {
	if len(d.registries) == 0 {
		return config.LoadDefaultConfigFile(os.Stderr)
	}

	auths := make(map[string]types.AuthConfig, len(d.registries))

	for host, credential := range d.registries {
		key := registryKey(host)
		auths[key] = types.AuthConfig{
			ServerAddress: key,
			Username:      credential.Username,
			Password:      credential.Password,
		}
	}

	return &configfile.ConfigFile{AuthConfigs: auths}
}

// registryKey is the name a credential is stored under. Docker Hub is the one
// registry whose key is not its host, and an empty host means it.
func registryKey(host string) string {
	switch host {
	case "", "docker.io", "index.docker.io", "registry-1.docker.io":
		return "https://index.docker.io/v1/"
	}
	return host
}
