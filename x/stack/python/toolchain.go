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

	// WithoutExtras leaves the optional dependency groups out.
	//
	// They are installed by default because a tool that cannot resolve an
	// import reports it as a fault of the code: on a repository whose tests
	// import its three extras, leaving them out invented eight type errors.
	// Set this for a project with an extra too heavy to install for a check
	// that does not read the code using it.
	WithoutExtras bool

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
	if t.WithoutExtras {
		return ""
	}
	return "--all-extras "
}
