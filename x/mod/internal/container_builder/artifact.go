package container_builder

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/client/llb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

// Artifact is the result of a successful build. It knows the recipe that
// produced it, which lets it deliver to another destination — a registry, the
// host disk — without rebuilding: BuildKit's content-addressed cache makes
// those redeliveries immediate.
type Artifact struct {
	builder *ContainerBuilder
	request BuildRequest
	image   *dockerspec.DockerOCIImage
}

// Config returns the produced image's configuration: entrypoint, user, ports,
// environment variables, labels. That is what makes it start.
func (a *Artifact) Config() *dockerspec.DockerOCIImage { return a.image }

// Push publishes the image to a registry under the given reference.
func (a *Artifact) Push(ctx context.Context, ref string) error {
	return a.deliverTo(ctx, a.toRegistry(ref), fmt.Sprintf("push %s", ref))
}

// PushInsecure publishes over plain HTTP, for an internal registry without TLS.
func (a *Artifact) PushInsecure(ctx context.Context, ref string) error {
	target := a.toRegistry(ref)
	target.entries[0].Attrs["registry.insecure"] = "true"
	return a.deliverTo(ctx, target, fmt.Sprintf("push %s", ref))
}

// SaveTar writes the image to an archive that `docker load` can read.
func (a *Artifact) SaveTar(ctx context.Context, ref, path string) error {
	return a.deliverTo(ctx, a.toTarball(ref, path), fmt.Sprintf("save tarball to %s", path))
}

// ExtractTo copies a path out of the image onto the host disk. This is how a
// stage collects its build artifacts: jars, binaries, reports.
// The source path accepts wildcards, which jar names need.
func (a *Artifact) ExtractTo(ctx context.Context, from, hostDir string) error {
	return a.deliverTo(ctx, a.toHostDir(from, hostDir), fmt.Sprintf("extract %s to %s", from, hostDir))
}

// ReadFile reads one file from the image without going through the disk.
// Useful to bring a value computed by the build back into the stage's Go code.
func (a *Artifact) ReadFile(ctx context.Context, path string) ([]byte, error) {
	daemon, err := a.builder.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer daemon.Close()

	return daemon.ReadFile(ctx, a.request, path)
}

// deliverTo is the common delivery path: open, solve, close.
func (a *Artifact) deliverTo(ctx context.Context, target exportTarget, operation string) error {
	daemon, err := a.builder.connect(ctx)
	if err != nil {
		return err
	}
	defer daemon.Close()

	if _, err := daemon.Solve(ctx, a.request, target); err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return nil
}

/* ─────────── The possible destinations ─────────── */

// exportTarget describes what the daemon must do with the result.
// Its zero value means export nothing: the graph is executed, that is all.
type exportTarget struct {
	entries []client.ExportEntry

	// subtree narrows the result to one path. Without it the local exporter
	// would dump the image's whole filesystem onto the disk.
	subtree string
}

// narrow reduces the result to the requested subtree. Wildcards are accepted:
// a Maven jar is named app-1.4.0-SNAPSHOT.jar, its exact name is unknown.
func (t exportTarget) narrow(state llb.State) llb.State {
	if t.subtree == "" {
		return state
	}
	return llb.Scratch().File(llb.Copy(state, t.subtree, "/", &llb.CopyInfo{
		AllowWildcard:  true,
		CreateDestPath: true,
	}))
}

func (a *Artifact) toRegistry(ref string) exportTarget {
	return exportTarget{entries: []client.ExportEntry{{
		Type: client.ExporterImage,
		Attrs: map[string]string{
			"name": ref,
			"push": "true",
		},
	}}}
}

// toTarball writes a stream: the docker exporter wants an io.WriteCloser, not
// a directory. The file is created when the daemon starts writing.
func (a *Artifact) toTarball(ref, path string) exportTarget {
	return exportTarget{entries: []client.ExportEntry{{
		Type:  client.ExporterDocker,
		Attrs: map[string]string{"name": ref},
		Output: func(map[string]string) (io.WriteCloser, error) {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return nil, fmt.Errorf("create directory for %s: %w", path, err)
			}
			return os.Create(path)
		},
	}}}
}

func (a *Artifact) toHostDir(from, dir string) exportTarget {
	return exportTarget{
		subtree: from,
		entries: []client.ExportEntry{{
			Type:      client.ExporterLocal,
			Attrs:     map[string]string{"dest": dir},
			OutputDir: dir,
		}},
	}
}
