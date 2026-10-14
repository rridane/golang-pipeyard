package common

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"github.com/rridane/golang-pipeyard/x/stack"
	"go.uber.org/zap"
)

// maxLineLength is how long a line may be before the word search gives up:
// without a ceiling, one line could be the whole file.
const maxLineLength = 1 << 20

// GuardStage holds a repository to its own rules about what may be committed:
// paths that must not appear, files that must not grow past a size, and words
// that must not be written down anywhere. It walks the files git would see and
// runs nothing in a container.
type GuardStage struct {
	// ForbiddenPaths are glob patterns, matched against a file's whole path
	// and against its base name: "*.ckpt" catches a name at any depth,
	// "data/raw/*" one directory's files. One segment at a time, as
	// path.Match does — there is no "**".
	ForbiddenPaths []string

	// MaxFileSize is the size in bytes above which a file is a violation.
	// Zero disables the check.
	MaxFileSize int64

	// ForbiddenWordsEnv names the variable holding the words no file may
	// contain, one per line or comma separated. A violation reports a word's
	// position in the list, never the word. Empty disables the check.
	ForbiddenWordsEnv string

	// WarnOnly reports the violations and succeeds, for adopting the stage on
	// a repository that does not pass it yet. It is meant to be removed.
	WarnOnly bool
}

// Violation is one rule broken by one file.
type Violation struct {
	Path   string
	Reason string
}

func (v Violation) String() string { return v.Path + ": " + v.Reason }

func (s *GuardStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("guard")
	spec.SetDescription("Hold the repository to its rules about what may be committed")
	return spec
}

func (s *GuardStage) Execute(
	ctx context.Context,
	args ru.ParsedArgs,
	sh *mod.ShellExecutor,
	log *zap.Logger,
) error {
	files, err := stack.TrackedFiles(ctx, sh)
	if err != nil {
		return err
	}

	words := ForbiddenWords(os.Getenv(s.ForbiddenWordsEnv))
	if s.ForbiddenWordsEnv != "" && len(words) == 0 {
		log.Warn("no word to look for", zap.String("variable", s.ForbiddenWordsEnv))
	}

	violations, err := s.Inspect(os.DirFS("."), files, words)
	if err != nil {
		return err
	}

	for _, v := range violations {
		log.Error("guard", zap.String("violation", v.String()))
	}

	switch {
	case len(violations) == 0:
		log.Info("every rule holds", zap.Int("files", len(files)))
		return nil
	case s.WarnOnly:
		log.Warn("rules broken, reported only", zap.Int("violations", len(violations)))
		return nil
	default:
		return fmt.Errorf("%d rule(s) broken", len(violations))
	}
}

// Inspect applies the rules, cheapest first. It takes an fs.FS so they are
// exercised without a repository, without git and without a disk.
func (s *GuardStage) Inspect(tree fs.FS, files, words []string) ([]Violation, error) {
	var found []Violation

	for _, file := range files {
		pattern, forbidden := s.forbiddenPath(file)

		if forbidden {
			// A file that should not be there at all is not worth reading.
			found = append(found, Violation{file, "matches the forbidden pattern " + pattern})
			continue
		}

		info, err := fs.Stat(tree, file)
		if errors.Is(err, fs.ErrNotExist) {
			continue // listed by git, already gone from the working tree
		}
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			continue
		}

		if s.MaxFileSize > 0 && info.Size() > s.MaxFileSize {
			found = append(found, Violation{file, fmt.Sprintf(
				"weighs %d bytes, over the %d allowed", info.Size(), s.MaxFileSize)})
			continue
		}

		if len(words) == 0 {
			continue
		}
		hits, err := forbiddenWordsIn(tree, file, words)
		if err != nil {
			return nil, err
		}
		found = append(found, hits...)
	}

	return found, nil
}

// forbiddenPath reports the first pattern a file matches.
func (s *GuardStage) forbiddenPath(file string) (pattern string, forbidden bool) {
	for _, p := range s.ForbiddenPaths {
		for _, candidate := range []string{file, path.Base(file)} {
			if ok, _ := path.Match(p, candidate); ok {
				return p, true
			}
		}
	}
	return "", false
}

// ForbiddenWords splits one environment value on newlines and commas, and
// lowercases it: the search is case-insensitive.
func ForbiddenWords(value string) []string {
	var words []string

	separator := func(r rune) bool { return r == '\n' || r == ',' }

	for _, raw := range strings.FieldsFunc(value, separator) {
		word := strings.ToLower(strings.TrimSpace(raw))

		if word != "" {
			words = append(words, word)
		}
	}
	return words
}

// forbiddenWordsIn reports every line holding one of the words, by the word's
// position: the stage's output is read by anyone with access to the job.
func forbiddenWordsIn(tree fs.FS, file string, words []string) ([]Violation, error) {
	f, err := tree.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var found []Violation

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), maxLineLength)

	for line := 1; scanner.Scan(); line++ {
		text := strings.ToLower(scanner.Text())
		for i, word := range words {
			if strings.Contains(text, word) {
				found = append(found, Violation{file, fmt.Sprintf(
					"holds forbidden word %d on line %d", i+1, line)})
			}
		}
	}

	err = scanner.Err()

	// A line too long to read is reported rather than ignored: it is also the
	// shape a file that should not be committed often has.
	if errors.Is(err, bufio.ErrTooLong) {
		return append(found, Violation{file, "holds a line too long to inspect"}), nil
	}
	if err != nil {
		return nil, err
	}

	return found, nil
}
