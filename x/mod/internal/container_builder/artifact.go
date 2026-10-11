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

type Artifact struct {
	builder *ContainerBuilder
	request BuildRequest
	image   *dockerspec.DockerOCIImage
}

func (a *Artifact) Config() *dockerspec.DockerOCIImage { return a.image }

func (a *Artifact) Push(ctx context.Context, ref string) error {
	return a.deliverTo(ctx, a.toRegistry(ref), fmt.Sprintf("push %s", ref))
}

// PushInsecure publishes over plain HTTP, for an internal registry without TLS.
func (a *Artifact) PushInsecure(ctx context.Context, ref string) error {
	target := a.toRegistry(ref)
	target.entries[0].Attrs["registry.insecure"] = "true"
	return a.deliverTo(ctx, target, fmt.Sprintf("push %s", ref))
}

func (a *Artifact) SaveTar(ctx context.Context, ref, path string) error {
	return a.deliverTo(ctx, a.toTarball(ref, path), fmt.Sprintf("save tarball to %s", path))
}

func (a *Artifact) ExtractTo(ctx context.Context, from, hostDir string) error {
	return a.deliverTo(ctx, a.toHostDir(from, hostDir), fmt.Sprintf("extract %s to %s", from, hostDir))
}

func (a *Artifact) ReadFile(ctx context.Context, path string) ([]byte, error) {
	daemon, err := a.builder.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer daemon.Close()

	return daemon.ReadFile(ctx, a.request, path)
}

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

// Its zero value means export nothing: the graph is executed, that is all.
type exportTarget struct {
	entries []client.ExportEntry

	// Without it the local exporter dumps the image's whole filesystem on disk.
	// Wildcards are accepted: a Maven jar name is not known in advance.
	subtree string
}

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
