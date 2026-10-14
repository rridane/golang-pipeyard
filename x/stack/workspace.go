package stack

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/rridane/golang-pipeyard/x/mod"
)

// ContainerRoot is where a workspace is mounted in every tool's image, and
// the working directory its script starts from.
const ContainerRoot = "/src"

// generatedDir keeps the files a stage produces apart from the repository's
// own, inside the workspace.
const generatedDir = ".pipeyard"

// DefaultOutDir is where a stage writes what it produces. It should be
// gitignored.
const DefaultOutDir = "out"

// Workspace is the build context a container stage receives: the files git
// would see, and nothing it ignores. A run from a laptop therefore carries no
// build output, no local state and no .env.
type Workspace struct {
	// Dir is where the files were copied, on the host.
	Dir string
}

// NewWorkspace copies the repository into a directory under outDir, replacing
// whatever a previous run left there. outDir itself is left out, so a
// workspace never contains an older one.
func NewWorkspace(ctx context.Context, sh *mod.ShellExecutor, outDir string) (Workspace, error) {
	files, err := TrackedFiles(ctx, sh)
	if err != nil {
		return Workspace{}, err
	}

	w := Workspace{Dir: filepath.Join(outDir, "workspace")}
	if err := os.RemoveAll(w.Dir); err != nil {
		return Workspace{}, fmt.Errorf("clear the previous workspace: %w", err)
	}

	excluded := outDir + "/"
	for _, name := range files {
		if strings.HasPrefix(name, excluded) {
			continue
		}
		if err := copyFile(name, filepath.Join(w.Dir, name)); err != nil {
			return Workspace{}, fmt.Errorf("copy %s into the workspace: %w", name, err)
		}
	}
	return w, nil
}

// Generated reserves a path for a file the stage produces: it creates the
// directory and hands back the host path and the container one.
func (w Workspace) Generated(name string) (host, inContainer string, err error) {
	host = filepath.Join(w.Dir, generatedDir, name)
	if err := os.MkdirAll(filepath.Dir(host), 0o755); err != nil {
		return "", "", err
	}
	return host, path.Join(ContainerRoot, generatedDir, name), nil
}

// write adds a generated file and returns the path the container will see.
func (w Workspace) write(name string, content []byte) (inContainer string, err error) {
	host, inContainer, err := w.Generated(name)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(host, content, 0o644); err != nil {
		return "", err
	}
	return inContainer, nil
}

// copyFile copies one file and keeps its permissions. A path git lists but
// which is gone from the working tree is skipped: the workspace is what the
// commit would contain.
func copyFile(from, to string) error {
	src, err := os.Open(from)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}

	dst, err := os.OpenFile(to, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}
