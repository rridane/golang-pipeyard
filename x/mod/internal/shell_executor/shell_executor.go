package shell_executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// ShellExecutor runs shell scripts through mvdan.cc/sh, without relying on a
// system shell being installed in the image.
type ShellExecutor struct {
	Timeout time.Duration // zero means no timeout
	Dir     string        // working directory; empty means the process one
}

// Run executes the script, streaming its output to stdout and stderr.
// It fails when the script cannot be parsed, exits non-zero, or exceeds Timeout.
func (s *ShellExecutor) Run(ctx context.Context, script string) error {
	return s.run(ctx, script, os.Stdout)
}

// Output executes the script and returns what it wrote on stdout. This is what
// lets a stage read a value computed by a command: a version from git describe,
// a revision from helm history, a resource from kubectl -o json.
func (s *ShellExecutor) Output(ctx context.Context, script string) ([]byte, error) {
	var stdout bytes.Buffer
	err := s.run(ctx, script, &stdout)
	return stdout.Bytes(), err
}

func (s *ShellExecutor) run(ctx context.Context, script string, stdout io.Writer) error {

	parsed, err := syntax.NewParser().Parse(strings.NewReader(script), "")
	if err != nil {
		return fmt.Errorf("parse script: %w", err)
	}

	runner, err := interp.New(s.runnerOptions(stdout)...)
	if err != nil {
		return fmt.Errorf("init shell runner: %w", err)
	}

	if s.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.Timeout)
		defer cancel()
	}

	if err := runner.Run(ctx, parsed); err != nil {
		return s.describeFailure(err)
	}

	return nil
}

func (s *ShellExecutor) runnerOptions(stdout io.Writer) []interp.RunnerOption {

	options := []interp.RunnerOption{interp.StdIO(os.Stdin, stdout, os.Stderr)}

	if s.Dir != "" {
		options = append(options, interp.Dir(s.Dir))
	}

	return options
}

// describeFailure turns a runner error into a message the pipeline author can act on.
func (s *ShellExecutor) describeFailure(err error) error {

	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("script timed out after %s", s.Timeout)
	}

	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("script cancelled: %w", err)
	}

	if status, ok := interp.IsExitStatus(err); ok {
		return fmt.Errorf("script exited with status %d", status)
	}

	return err
}
