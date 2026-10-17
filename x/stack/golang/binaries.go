package golang

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"github.com/rridane/golang-pipeyard/x/stack"
	"go.uber.org/zap"
)

// manifestName is the file binaries writes and publish-binaries reads.
const manifestName = "binaries.json"

// checksumsName lists the changed binaries for the attestation.
const checksumsName = "checksums.txt"

// BinariesStage builds every command of the repository and keeps those that
// differ from their last release. Each command is released on its own, under
// tags of its own, "name/v1.2.3". The builds are reproducible, so comparing
// digests answers whether a binary changed, whatever the cause: its own code,
// a shared package or a dependency.
type BinariesStage struct {
	// Commands holds one directory per binary. Empty uses "cmd".
	Commands string

	// Image is the Go toolchain, pinned. Empty uses DefaultGoImage.
	Image string

	// OS and Arch are the target. Empty uses linux and amd64.
	OS, Arch string

	// OutDir is where the stage may write. Empty uses stack.DefaultOutDir.
	OutDir string
}

// Binary is one command that changed, with the tag it is released under.
type Binary struct {
	Name   string `json:"name"`
	Tag    string `json:"tag"`
	SHA256 string `json:"sha256"`
}

func (s *BinariesStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("binaries")
	spec.SetDescription("Build the commands and keep those that changed since their release")
	spec.AddFlag("dist", ru.StringFlag, "directory the changed binaries are written to", "./dist", false)
	return spec
}

func (s *BinariesStage) Execute(
	ctx context.Context,
	args ru.ParsedArgs,
	sh *mod.ShellExecutor,
	cb *mod.ContainerBuilder,
	releases *mod.GithubReleaseSDK,
	log *zap.Logger,
) error {
	dir := cmp.Or(s.Commands, "cmd")
	names, err := commands(dir)
	if err != nil {
		return err
	}

	workspace, err := stack.NewWorkspace(ctx, sh, cmp.Or(s.OutDir, stack.DefaultOutDir))
	if err != nil {
		return err
	}

	dist := args.String("dist")
	if err := os.RemoveAll(dist); err != nil {
		return fmt.Errorf("clear %s: %w", dist, err)
	}

	built, err := cb.Build(ctx, mod.BuildRequest{Dockerfile: s.recipe(dir, names), ContextDir: workspace.Dir})
	if err != nil {
		return fmt.Errorf("build the binaries: %w", err)
	}
	if err := built.ExtractTo(ctx, "/out/*", dist); err != nil {
		return fmt.Errorf("extract the binaries: %w", err)
	}

	changed := []Binary{}

	for _, name := range names {
		path := filepath.Join(dist, name)
		digest, err := fileDigest(path)
		if err != nil {
			return err
		}

		binary, err := decide(ctx, releases, name, digest)
		if err != nil {
			return err
		}
		if binary == nil {
			log.Info("unchanged since its release", zap.String("binary", name))
			if err := os.Remove(path); err != nil {
				return err
			}
			continue
		}

		checksum := fmt.Sprintf("%s  %s\n", digest, name)
		if err := os.WriteFile(path+".sha256", []byte(checksum), 0o644); err != nil {
			return err
		}
		log.Info("changed", zap.String("binary", name), zap.String("tag", binary.Tag))
		changed = append(changed, *binary)
	}

	// What the CI attests before anything is published, in the format
	// sha256sum writes and actions/attest-build-provenance reads.
	var checksums strings.Builder
	for _, binary := range changed {
		fmt.Fprintf(&checksums, "%s  %s\n", binary.SHA256, binary.Name)
	}
	if err := os.WriteFile(filepath.Join(dist, checksumsName), []byte(checksums.String()), 0o644); err != nil {
		return err
	}

	manifest, err := json.MarshalIndent(changed, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dist, manifestName), manifest, 0o644); err != nil {
		return err
	}

	log.Info("binaries built", zap.Int("commands", len(names)), zap.Int("changed", len(changed)))
	return nil
}

// recipe builds every command in one image. What makes two builds of the same
// source identical: no cgo, no paths of the machine, no VCS stamp, no build id.
func (s *BinariesStage) recipe(dir string, names []string) mod.Dockerfile {
	return mod.Dockerfile{
		Args: struct {
			Image, OS, Arch, Dir string
			Names                []string
		}{cmp.Or(s.Image, DefaultGoImage), cmp.Or(s.OS, "linux"), cmp.Or(s.Arch, "amd64"), dir, names},
		Template: `
			FROM {{.Image}}
			WORKDIR /src
			COPY go.mod go.sum ./
			RUN --mount=type=cache,target=/go/pkg/mod go mod download
			COPY . .
			ENV CGO_ENABLED=0 GOOS={{.OS}} GOARCH={{.Arch}} GOFLAGS=-trimpath
			RUN --mount=type=cache,target=/go/pkg/mod \
			    --mount=type=cache,target=/root/.cache/go-build \
			    {{range .Names}}go build -buildvcs=false -ldflags "-s -w -buildid=" -o /out/{{.}} ./{{$.Dir}}/{{.}} && {{end}}true
		`,
	}
}

// releaseFinder is the part of the forge decide reads, so that a test can
// stand in for GitHub.
type releaseFinder interface {
	PublishedWithPrefix(ctx context.Context, prefix string) ([]mod.GithubPublishedRelease, error)
	DownloadAsset(ctx context.Context, id int64) ([]byte, error)
}

// decide compares a binary with its last release. It returns nil when they are
// the same, and otherwise the tag the binary is released under next: the last
// one with its patch raised, or v0.1.0 for a binary never released.
func decide(ctx context.Context, releases releaseFinder, name, digest string) (*Binary, error) {
	published, err := releases.PublishedWithPrefix(ctx, name+"/v")
	if err != nil {
		return nil, err
	}

	last, version, ok := latest(published, name)
	if !ok {
		return &Binary{Name: name, Tag: name + "/v0.1.0", SHA256: digest}, nil
	}

	if id, found := last.Assets[name+".sha256"]; found {
		checksum, err := releases.DownloadAsset(ctx, id)
		if err != nil {
			return nil, err
		}
		if fields := strings.Fields(string(checksum)); len(fields) > 0 && fields[0] == digest {
			return nil, nil
		}
	}

	version[2]++
	tag := fmt.Sprintf("%s/v%d.%d.%d", name, version[0], version[1], version[2])
	return &Binary{Name: name, Tag: tag, SHA256: digest}, nil
}

// latest picks the highest version among the releases of one binary. A tag
// that is not name/vX.Y.Z is left out rather than guessed at.
func latest(published []mod.GithubPublishedRelease, name string) (mod.GithubPublishedRelease, [3]int, bool) {
	type candidate struct {
		release mod.GithubPublishedRelease
		version [3]int
	}
	var candidates []candidate

	for _, release := range published {
		if version, ok := parseVersion(strings.TrimPrefix(release.Tag, name+"/v")); ok {
			candidates = append(candidates, candidate{release, version})
		}
	}
	if len(candidates) == 0 {
		return mod.GithubPublishedRelease{}, [3]int{}, false
	}

	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i].version, candidates[j].version
		for k := range a {
			if a[k] != b[k] {
				return a[k] > b[k]
			}
		}
		return false
	})
	return candidates[0].release, candidates[0].version, true
}

func parseVersion(text string) ([3]int, bool) {
	parts := strings.Split(text, ".")
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var version [3]int
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return [3]int{}, false
		}
		version[i] = n
	}
	return version, true
}

// commands lists the directories under dir, one per binary.
func commands(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("list the commands in %s: %w", dir, err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no command in %s", dir)
	}
	return names, nil
}

func fileDigest(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}
