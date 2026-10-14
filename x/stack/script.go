package stack

import (
	"fmt"
	"strings"
)

// Script builds a shell script that runs every step even when one fails, and
// exits non-zero if any of them did. A check that stops at the first failure
// makes its reader run it again to see the rest.
type Script struct {
	steps []string
}

// Step adds a command, formatting its arguments.
func (s *Script) Step(format string, args ...any) *Script {
	s.steps = append(s.steps, fmt.Sprintf(format, args...))
	return s
}

// String renders the script. Every step announces itself, so the output says
// which command produced what.
func (s *Script) String() string {
	var b strings.Builder

	b.WriteString("status=0\n")
	for _, step := range s.steps {
		fmt.Fprintf(&b, "echo %s\n%s || status=1\n", Quote("== "+step), step)
	}
	b.WriteString("exit $status\n")

	return b.String()
}
