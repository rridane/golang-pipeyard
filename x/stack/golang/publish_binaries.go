package golang

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"go.uber.org/zap"
)

// PublishBinariesStage releases what binaries kept. It is a stage of its own so
// that the CI can attest the binaries in between: a release is only published
// once its binary can be proven to come from this repository's workflow.
type PublishBinariesStage struct{}

func (s *PublishBinariesStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("publish-binaries")
	spec.SetDescription("Release every binary that changed, each under its own tag")
	spec.AddFlag("dist", ru.StringFlag, "directory binaries wrote to", "./dist", false)
	return spec
}

func (s *PublishBinariesStage) Execute(
	ctx context.Context,
	args ru.ParsedArgs,
	identity pi.Identity,
	releases *mod.GithubReleaseSDK,
	log *zap.Logger,
) error {
	if identity.CommitSHA == "" {
		return fmt.Errorf("publish-binaries tags the commit it runs on, and %s is empty", pi.CommitSHAEnvVar)
	}

	dist := args.String("dist")
	binaries, err := readManifest(dist)
	if err != nil {
		return err
	}
	if len(binaries) == 0 {
		log.Info("no binary changed, nothing to release")
		return nil
	}

	for _, binary := range binaries {
		path := filepath.Join(dist, binary.Name)

		// A draft until both files are attached: a release becomes visible to
		// the consumers the moment it is published.
		published := releases.
			WithTag(binary.Tag).
			WithReleaseName(binary.Tag).
			WithTargetBranch(identity.CommitSHA).
			WithBody(fmt.Sprintf("sha256 `%s`", binary.SHA256)).
			AsDraft(true).
			PrepareRelease().
			CreateTag().
			CreateRelease().
			UploadAssets([]string{path, path + ".sha256"}).
			Publish()

		if err := published.Err(); err != nil {
			return fmt.Errorf("release %s: %w", binary.Tag, err)
		}
		log.Info("released", zap.String("tag", binary.Tag), zap.String("sha256", binary.SHA256))
	}
	return nil
}

func readManifest(dist string) ([]Binary, error) {
	raw, err := os.ReadFile(filepath.Join(dist, manifestName))
	if err != nil {
		return nil, fmt.Errorf("read the manifest binaries wrote: %w", err)
	}
	var binaries []Binary
	if err := json.Unmarshal(raw, &binaries); err != nil {
		return nil, fmt.Errorf("the manifest in %s is not valid: %w", dist, err)
	}
	return binaries, nil
}
