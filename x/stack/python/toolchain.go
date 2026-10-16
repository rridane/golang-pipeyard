package python

import (
	"cmp"
	"strings"

	"github.com/rridane/golang-pipeyard/x/stack"
)

// uvCache is where uv keeps what it downloads, asked of the image rather than
// assumed.
const uvCache = "/root/.cache/uv"

// Toolchain is how a stage reaches the project's own tools: uv restores what
// the lockfile pins, so the version a stage runs is the version the project
// declares. One source of truth instead of two.
//
// The image is Debian rather than Alpine because some tools ship a binary of
// their own — pyright carries a Node — and those are linked against glibc.
type Toolchain struct {
	// Image is where uv is installed. Empty uses DefaultImage.
	Image string

	// UvVersion is installed into Image. Empty uses DefaultUvVersion.
	UvVersion string

	// Packages are Debian packages to install before uv, for a tool whose
	// own binary needs a library the slim image leaves out.
	Packages []string

	// AllExtras installs every optional dependency group. A stage whose work
	// reads the code that imports them needs it: without, the tool reports
	// an unresolved import as a fault of the repository.
	AllExtras bool

	// Caches are container paths to keep between runs, besides uv's own.
	Caches []string
}

// Tool runs one command through uv, which restores the lockfile's versions
// before the command starts.
func (t Toolchain) Tool(name, command string) stack.Tool {
	return stack.Tool{
		Name:   name,
		Image:  cmp.Or(t.Image, DefaultImage),
		Setup:  t.setup(),
		Script: "uv run --frozen " + t.extras() + command + "\n",
		Caches: append([]string{uvCache}, t.Caches...),
	}
}

// setup installs the Debian packages a tool's own binary needs, then uv. The
// two are one RUN so the image carries one layer instead of three.
func (t Toolchain) setup() string {
	install := "pip install --no-cache-dir uv==" + cmp.Or(t.UvVersion, DefaultUvVersion)

	if len(t.Packages) == 0 {
		return "RUN " + install
	}

	return "RUN apt-get update" +
		" && apt-get install -y --no-install-recommends " + strings.Join(t.Packages, " ") +
		" && rm -rf /var/lib/apt/lists/*" +
		" && " + install
}

func (t Toolchain) extras() string {
	if t.AllExtras {
		return "--all-extras "
	}
	return ""
}
