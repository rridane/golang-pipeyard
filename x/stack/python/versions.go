package python

// Every tool is pinned here, so a laptop and the CI run the same version, and
// each was published at least two weeks before it was pinned. A project that
// declares one of these itself should pin the same version in both places.
const (
	// DefaultImage is where the tools are installed. Debian rather than
	// Alpine: pyright ships its own Node, linked against glibc.
	DefaultImage = "python:3.12-slim"

	// DefaultRuffVersion is the linter and formatter.
	DefaultRuffVersion = "0.16.9"

	// DefaultUvVersion resolves and verifies the lockfiles.
	DefaultUvVersion = "0.11.3"
)
