# 🧱 Pipewire – Centralized, Observable CI Pipelines for Go

> **Imagine a pipeline as clean as this for your wonderful project.**  
> You define your stages in Go.  
> Your workflows run everywhere — GitHub Actions, GitLab CI, Jenkins, Dagger...  
> All fully instrumented with OpenTelemetry and auto-wired dependencies.

---

## 🚀 Example – The dream CI

Let’s imagine your project `mywonderfulproject-cli`, a small Go command-line tool.  
Here’s how its **GitHub Actions pipeline** looks when powered by Pipewire:

```yaml
name: 🚀 MyWonderfulProject CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]
  workflow_dispatch:

jobs:
  pipeline:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: [ '1.22', '1.23' ]

    steps:
      - name: 📦 Checkout
        uses: actions/checkout@v4

      - name: 🧰 Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go-version }}
          cache: true

      - name: 🧪 Run tests via Pipewire
        run: go run ./ci test --timeout 60s

      - name: ⚙️ Validate build via Pipewire
        run: go run ./ci build
```

That’s it — just **two stages**.  
Everything else (logging, tracing, metrics, dependency wiring, OpenTelemetry export…) is handled by Pipewire.

---

## ✨ How it works

Pipewire turns your Go project into a **self-contained CI pipeline**.  
Each step (`stage`) is a Go struct implementing a simple contract.  
Pipewire handles orchestration, dependency injection, and observability automatically.

### Stage definition

A stage is any Go struct that implements:

```go
type Stage interface {
  Configure(cmd *cobra.Command)
  Execute(ctx context.Context, cmd *cobra.Command, /* your injected deps */) error
}
```

Example:  
```go
type BuildStage struct{}

func (s *BuildStage) Configure(cmd *cobra.Command) {
  cmd.Use = "build"
  cmd.Short = "Validate Go module buildability"
}

func (s *BuildStage) Execute(ctx context.Context, cmd *cobra.Command, logger *zap.Logger, shell *shell_executor.ShellExecutor) error {
  logger.Info("Building module...")
  return shell.Run(ctx, "go build ./...")
}
```

You register it in your pipeline entrypoint:

```go
pipe.
  RegisterStage("test", &TestStage{}).
  RegisterStage("build", &BuildStage{})

pipe.Run()
```

When you run `go run ./ci build`, Pipewire:
- parses CLI args via Cobra,
- automatically injects all required dependencies,
- starts an OpenTelemetry span + metrics counters,
- runs your stage code,
- and exports all results (logs, traces, metrics) to your configured OTel endpoint.

---

## 📜 Command system (Cobra-powered)

Every **Pipewire pipeline** is a real **command-line application**, powered by [Cobra](https://github.com/spf13/cobra) — the same library used by `kubectl`, `docker`, or `helm`.

Each stage you register automatically becomes a **Cobra command** with:
- its own description (`Use`, `Short`, `Long`),
- its own flags (defined in `Configure()`),
- built-in help (`--help`),
- and fully typed arguments (`Args: cobra.ExactArgs(n)`).

### Example

```go
func (s *ReleaseStage) Configure(cmd *cobra.Command) {
  cmd.Use = "release <version>"
  cmd.Short = "Publish a new GitHub release"
  cmd.Long = "Creates and publishes a new release on GitHub with changelog and assets."
  cmd.Args = cobra.ExactArgs(1)
  cmd.Flags().Bool("prerelease", false, "Mark release as pre-release")
  cmd.Flags().Bool("publish", true, "Publish the release instead of draft")
}
```

Now, running:
```bash
go run ./ci release --help
```

Produces:
```
Usage:
  ci release <version> [flags]

Flags:
      --prerelease   Mark release as pre-release
      --publish       Publish the release instead of draft
  -h, --help          Help for release
```

✅ You don’t need to handle `os.Args` or flag parsing — Pipewire automatically creates and links your commands,  
generates the CLI help, and binds them to their corresponding stages.

Behind the scenes:
- `PipelineProvider.New()` builds a root Cobra command (your main CLI).
- Each call to `RegisterStage("name", stage)` attaches a subcommand.
- When you run `go run ./ci <stage>`, Cobra executes the correct handler and Pipewire injects dependencies automatically.

This makes Pipewire not only a **CI orchestrator**, but also a **CLI framework** for defining instrumented pipelines.

---

## ⚙️ Dependency injection – zero boilerplate

Each stage declares **only what it needs** in its `Execute()` parameters.  
Pipewire resolves and injects everything at runtime.

### Example
```go
func (s *TestStage) Execute(
  ctx context.Context,
  cmd *cobra.Command,
  logger *zap.Logger,
  shell *shell_executor.ShellExecutor,
  meter metric.Meter,
  tracer *trace.Tracer,
) error { ... }
```

No setup, no wiring, no globals.  
Pipewire detects each argument’s type and automatically provides it.

---

## 🧩 Built-in injectors

| Injector | Type Injected | Description | Env vars |
|-----------|----------------|--------------|-----------|
| **LoggerInjector** | `*zap.Logger` | Structured JSON logger with optional OTel export | `OTEL_EXPORTER_OTLP_ENDPOINT`, `LOG_LEVEL` |
| **TracerInjector** | `*trace.Tracer` | OpenTelemetry tracer for spans | `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_SERVICE_NAME` |
| **MeterInjector** | `metric.Meter` | OTel metrics provider | `OTEL_EXPORTER_OTLP_ENDPOINT` |
| **ShellExecutorInjector** | `*ShellExecutor` | Safe command runner (stdout/stderr + context) | — |
| **GitHubReleaseInjector** | `*github_releases.ReleaseFactory` | High-level SDK for GitHub release management | `GITHUB_TOKEN`, `GITHUB_OWNER`, `GITHUB_REPO` |
| **DaggerInjector** | `*dagger.Client` | Dagger client for containerized builds | `DAGGER_CLOUD_TOKEN` |
| **K8sInjector** | `*kubernetes.Clientset` | Access to Kubernetes cluster (for CD use cases) | `KUBECONFIG`, `K8S_NAMESPACE` |

### 💡 Recommended configuration practice

Pipewire recommends configuring injectors **using environment variables** at initialization time.  
This ensures reproducibility across local development, CI pipelines, and production environments.

Example:

```go
&injectors.LoggerInjector{
  ServiceName:              os.Getenv("SERVICE_NAME"),
  EnableOtelExport:         true,
  OtelExporterOtlpEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
  LogLevel:                 os.Getenv("LOG_LEVEL"),
}
```

💬 This makes the pipeline portable and CI-friendly:  
GitHub Actions, Jenkins, or Dagger can simply set environment variables, and your injectors automatically adapt.

---

## 🧠 What’s an “Injector”?

An **Injector** is a small struct responsible for configuring and providing a dependency.  
It can wrap any internal module or third-party library (Zap, OpenTelemetry, Kubernetes, Dagger…).

Each injector exposes a single public method:

```go
type LoggerInjector struct {
  ServiceName string
  LogLevel string
  EnableOtelExport bool
  OtelExporterOtlpEndpoint string
}

func (l *LoggerInjector) Inject(stageName string) *zap.Logger { ... }
```

When a stage declares a `*zap.Logger` argument,  
Pipewire calls all injectors until one returns a non-nil value of that type.

This gives you:
- per-stage configuration,
- no global state,
- predictable, testable dependencies.

---

## 🔧 Create your own injector

To make your pipeline plug-in friendly, you can define your own injectors.

Example:  
Injecting a configured HTTP client with retry + metrics:

```go
type HttpClientInjector struct {
  Timeout time.Duration
}

func (i *HttpClientInjector) Inject(stageName string) *http.Client {
  return &http.Client{
    Timeout: i.Timeout,
  }
}
```

Then register it in your pipeline:

```go
pipe := pipeline_provider.New(pipeline_provider.PipelineConfig{
  Name: "ci_pipeline",
  Injectors: []any{
    &injectors.LoggerInjector{...},
    &injectors.TracerInjector{...},
    &HttpClientInjector{Timeout: 5 * time.Second},
  },
})
```

Now, any stage can simply declare:

```go
func (s *ApiTestStage) Execute(ctx context.Context, client *http.Client, logger *zap.Logger) error {
  resp, _ := client.Get("https://api.example.com/health")
  logger.Info("Response received", zap.Int("status", resp.StatusCode))
  return nil
}
```

No setup. No configuration duplication.

---

## 🧭 Full working example

### Directory structure
```
ci/
├── main.go
├── test_stage.go
├── build_stage.go
├── release_stage.go
```

### `main.go`
```go
package main

import (
  "github.com/rridane/golang-pipewire"
  "github.com/rridane/golang-pipewire/injectors"
)

func main() {
  pipe := pipeline_provider.New(pipeline_provider.PipelineConfig{
    Name: "ci_pipeline",
    Version: "1.0.0",
    Injectors: []any{
      &injectors.LoggerInjector{ServiceName: "pipewire"},
      &injectors.TracerInjector{ServiceName: "pipewire"},
      &injectors.ShellExecutorInjector{},
      &injectors.MeterInjector{ServiceName: "pipewire"},
    },
  })

  pipe.
    RegisterStage("test", &TestStage{}).
    RegisterStage("build", &BuildStage{}).
    RegisterStage("release", &ReleaseStage{})

  pipe.Run()
}
```

### Run locally
```bash
go run ./ci test
go run ./ci build
go run ./ci release v1.0.0 --publish
```

### Or run in GitHub Actions
```yaml
- name: 🧪 Test
  run: go run ./ci test

- name: ⚙️ Build
  run: go run ./ci build

- name: 🚀 Release
  if: startsWith(github.ref, 'refs/tags/')
  run: go run ./ci release ${GITHUB_REF#refs/tags/} --publish
```

---

## 📊 Observability baked-in

Every stage automatically emits:
- **Traces** (OpenTelemetry spans)
- **Metrics** (durations, counts)
- **Logs** (structured JSON with context)

You can connect to any OTLP endpoint:  
Prometheus, Grafana Cloud, SigNoz, Datadog, Tempo, or your own collector.

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
```

All data is correlated per stage (`test`, `build`, `release`).

---

## 🧠 Summary

✅ Centralized orchestration of all your pipelines  
✅ Same CLI and CI syntax (`go run ./ci <stage>`)  
✅ Automatic dependency injection  
✅ Built-in observability  
✅ Extensible with your own injectors  
✅ Portable across any CI/CD system  

---

## 🌀 Self-hosted pipeline

Pipewire is self-powered: it **uses its own pipeline** to run its tests, build validations, and GitHub releases.  
You can explore the folder [`ci/`](./ci) inside this repository to see how its own stages (`test`, `build`, `release`) are implemented — and use it as a reference to build your own.
