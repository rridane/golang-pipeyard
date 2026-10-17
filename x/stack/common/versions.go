package common

// DefaultGitleaksImage carries the secret scanner, pinned so a laptop and the
// CI run the same version.
const DefaultGitleaksImage = "zricethezav/gitleaks:v8.30.1"

// DefaultZizmorImage carries the workflow auditor, pinned for the same reason.
const DefaultZizmorImage = "ghcr.io/zizmorcore/zizmor:1.30.1"
