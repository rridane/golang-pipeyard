package stack

import "strings"

// Quote wraps a value for a shell script, so a path with a space or a
// parenthesis survives.
func Quote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
