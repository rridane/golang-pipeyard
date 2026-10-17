package golang

// DefaultLinterImage carries golangci-lint, pinned so a laptop and the CI
// report the same findings.
const DefaultLinterImage = "golangci/golangci-lint:v2.13.2-alpine"

// DefaultPackages is the package pattern every stage falls back to.
const DefaultPackages = "./..."

// DefaultGoImage compiles the released binaries. It is pinned to a patch
// release: a binary is only reproducible under the same toolchain, and a
// toolchain that moved would make every binary look changed.
const DefaultGoImage = "golang:1.26.8-alpine"
