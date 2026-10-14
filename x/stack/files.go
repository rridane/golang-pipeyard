package stack

import (
	"context"
	"fmt"
	"strings"

	"github.com/rridane/golang-pipeyard/x/mod"
)

// TrackedFiles lists the files git would see: the ones it follows, and the new
// ones it is not ignoring. An ignored file never reaches a commit, so a check
// about what may be committed walks exactly those. Paths are relative to the
// repository root, with forward slashes.
func TrackedFiles(ctx context.Context, sh *mod.ShellExecutor) ([]string, error) {
	listing, err := sh.Output(ctx, "git ls-files -z --cached --others --exclude-standard")
	if err != nil {
		return nil, fmt.Errorf("list the repository files: %w", err)
	}

	names := strings.Split(string(listing), "\x00")

	files := make([]string, 0, len(names))
	for _, name := range names {
		if name != "" {
			files = append(files, name)
		}
	}
	return files, nil
}
