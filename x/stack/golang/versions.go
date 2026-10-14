package golang

// DefaultLinterImage carries golangci-lint, pinned so a laptop and the CI
// report the same findings.
const DefaultLinterImage = "golangci/golangci-lint:v1.62-alpine"

// DefaultPackages is the package pattern every stage falls back to.
const DefaultPackages = "./..."
