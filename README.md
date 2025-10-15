# 🧱 PipeYard — Centralized, Cross-Platform CI Orchestration in Go

> **Imagine a pipeline as clean as this for your Go project.**
>
> You define your stages in Go.  
> Your workflows run everywhere — GitHub Actions, GitLab CI, Azure Devops, Jenkins, Dagger...  
> All fully instrumented with OpenTelemetry and auto-wired dependencies.
---

# 🚀 Example

Let’s imagine your project mywonderfuljavaproject-cli, a small Go-powered CLI that drives your Java build system.

Here’s what your perfect pipeline looks like before plugging it into GitHub Actions, Azure DevOps, or Jenkins:

```bash
# 🏗️  1. Clean & Prepare Environment
mywonderfuljavaproject-cli prepare --maven-version=3.9.6

# 🧪  2. Run Unit & Integration Tests
mywonderfuljavaproject-cli test --reports-dir=build/reports/tests --timeout=5m

# 🔍  3. Run Static Analysis (Lint + SCA)
mywonderfuljavaproject-cli lint --tools=spotbugs

# ⚙️  4. Build Artifact (JAR + Docker)
mywonderfuljavaproject-cli build --dockerfile=Dockerfile --image=ghcr.io/myorg/mywonderfulproject:x.x.x --platform=linux/amd64,linux/arm64

# 📦  5. Publish Artifact to Registry
mywonderfuljavaproject-cli publish --image=ghcr.io/myorg/mywonderfulproject:latest --push=true --sign=true

# ☸️  6. Deploy to Kubernetes
mywonderfuljavaproject-cli deploy --cluster=staging --namespace=mywonderfulproject --manifest=k8s/deployment.yaml

# ======================================
mywonderfuljavaproject-cli verify --smoke=true --endpoint=https://staging.mywonderfulproject.internal/healthz --timeout=2m
```

And then, you can use it as-is inside your favorite CI provider — no duplicate logic, no YAML drift:

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
    steps:
      - name: 📦 Checkout
        uses: actions/checkout@v4

      - name: 🧰 Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      # Run tests via your unified PipeYard-based CLI
      - name: 🧪 Run tests
        run: mywonderfuljavaproject-cli test --scope=unit,integration--coverage-threshold=80 --reports-dir=build/reports/tests --timeout=5m

      # Build via the same unified CLI (re-uses the exact Go pipeline)
      - name: ⚙️ Build
        run: mywonderfuljavaproject-cli build --module=mywonderfulproject-service --dockerfile=Dockerfile --image=ghcr.io/myorg/mywonderfulproject:x.x.x --platform=linux/amd64,linux/arm64
      #...
```

Now imagine you could simply declare each stage in Go — define its parameters, its inputs and outputs — and PipeYard would automatically compile this CLI for you.

That’s exactly what **PipeYard** enables 

> a **unified, observable, cross‑CI pipeline**, deployable anywhere.

Let's dive in.

## ✨ How it works

pipeyard turns your Go project into a **self-contained, observable CI pipeline**.

Each stage (`test`, `build`, `release`) is a Go struct implementing a simple contract.  
pipeyard handles orchestration, dependency injection, and full OpenTelemetry observability.

---

## ⚙️ Stage definition

A stage is any Go struct implementing:

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

## 🧩 Assembling the Pipeline

Once your stages are defined, pipeyard lets you **assemble them into a real command-line pipeline**.

You do this in a `main.go` file — typically in a `ci/` folder — where you create a pipeline and register your stages.


Example:

> **_NOTE:_** We will explain in detail injectors

```go
func main() {

    ctx := context.Background()

    err := godotenv.Overload(".env")

    if err != nil {
        log.Println("not found .env, use env variables")
    }

    // 🧭 Configure injectors
    tracerInjector, telemetryLoggerInjector, meterInjector := getInjectors()

    pipe := pipeline_provider.New(pipeline_provider.PipelineConfig{
        Name:             "ci_pipeline",
        ShortDescription: "CI CLI – All stages",
        LongDescription:  "Pipeline cli for pipeyard library",
        Version:          "1.0.0",
        Injectors: []any{
            &injectors.ShellExecutorInjector{},
            &tracerInjector,
            &telemetryLoggerInjector,
            &meterInjector,
        },
    })

    // 🧩 Register stages
    pipe.
        RegisterStage("test", &TestSTage{}).
        RegisterStage("build", &BuildStage{}).
        RegisterStage("release", &ReleaseStage{})

    pipe.Run()

    // 🧹 Gracefully shutdown telemetry providers
    tracerInjector.Shutdown(ctx)
    telemetryLoggerInjector.Shutdown(ctx)
    meterInjector.Shutdown(ctx)

}

func getInjectors() (injectors.TracerInjector, injectors.MeterInjector, injectors.TelemetryLoggerInjector) {

    tracerInjector := injectors.TracerInjector{
        ServiceName:              "pipeyard",
        OtelExporterOtlpEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
    }

    telemetryLoggerInjector := injectors.TelemetryLoggerInjector{
        ServiceName:              "pipeyard",
        EnableOtelExport:         true,
        OtelExporterOtlpEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
        LogLevel:                 zapcore.InfoLevel,
    }

    meterInjector := injectors.MeterInjector{
        ServiceName:              "pipeyard",
        OtelExporterOtlpEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
    }

    return tracerInjector, meterInjector, telemetryLoggerInjector

}
```

✅ This cleanly declares a complete pipeline:
- with automatic injection of logger, tracer, and meter,
- with OpenTelemetry export,
- and proper cleanup at the end.


## 🧠 Cobra Integration

Because pipeyard is powered by [Cobra](https://github.com/spf13/cobra),  
you instantly get a **full CLI interface** for your pipeline.

### Root help

```bash
go run ./ci --help
```

Output:

```
CI CLI – All stages

Usage:
  ci [command]

Available Commands:
  test        Run the test suite
  build       Validate Go module buildability
  release     Create and publish a GitHub release

Flags:
  -h, --help      help for ci
  --version       show version info
```

### Command-specific help

```bash
go run ./ci build --help
```

Output:

```
Usage:
  ci build [flags]

Flags:
  -h, --help   help for build
```

✅ You don’t need to manually parse or manage flags — pipeyard + Cobra handle everything.  
Each `RegisterStage` automatically becomes a subcommand, fully typed and documented.

🧩 At this point, your pipeline is fully assembled — all your stages are defined, your CLI is automatically generated, and your CI is ready to run anywhere (locally, in GitHub Actions, GitLab CI, Jenkins, or even inside a Dagger pipeline).

## 🧠 Injectors – The heart of pipeyard's Dependency Injection

Injectors are the **core mechanism** that power pipeyard’s dependency injection system.

An **injector** is a simple Go `struct` that provides a ready-to-use instance of a module or service (e.g. logger, tracer, Kubernetes client) with its own configuration.  
In other words:

> 💡 **Injector = Module + Specific Configuration — supercharged for automatic injection.**

You don’t have to manually pass dependencies between stages.  
pipeyard automatically matches the parameters of each stage’s `Execute()` function to the appropriate injectors.

For example:

```go
type MyStage struct{}

func (s *MyStage) Execute(ctx context.Context, cmd *cobra.Command, logger *zap.Logger, tracer trace.Tracer) error {
  logger.Info("Running MyStage...")
  ctx, span := tracer.Start(ctx, "MyStage")
  defer span.End()
  return nil
}
```

As long as your pipeline includes a `TelemetryLoggerInjector` and a `TracerInjector`, pipeyard automatically injects both dependencies into this stage.

## ⚙️ Built-in Injectors

| Injector | Type | Description | Shutdown required | Injected Properties |
|-----------|------|--------------|-------------------|---------------------|
| `TelemetryLoggerInjector` | `*zap.Logger` | Structured JSON + OTLP logs (Zap + OpenTelemetry Logs bridge) | ✅ | `ServiceName`, `EnableOtelExport`, `OtelExporterOtlpEndpoint`, `LogLevel`, `OtelInsecure` |
| `TracerInjector` | `trace.Tracer` | OpenTelemetry tracing | ✅ | `ServiceName`, `OtelExporterOtlpEndpoint`, `Insecure` |
| `MeterInjector` | `metric.Meter` | OpenTelemetry metrics | ✅ | `ServiceName`, `OtelExporterOtlpEndpoint`, `Insecure` |
| `ShellExecutorInjector` | `*ShellExecutor` | Context-aware shell runner (built on `mvdan/sh`) | ❌ | `Timeout` |
| `GitHubReleaseInjector` | `*github_releases.ReleaseFactory` | Manage GitHub releases programmatically | ❌ | `Token`, `Owner`, `Repo`, `Enterprise`, `BaseURL` |
| `K8sInjector` | `*kubernetes.Clientset` | Access to Kubernetes cluster using in-cluster or kubeconfig context | ❌ | `KubeConfigPath`, `Namespace` |
| `DaggerInjector` | `*dagger.Client` | Run containerized CI/CD pipelines inside Dagger Engine | ❌ | `Endpoint`, `AuthToken`, `Insecure` |

💡 **Best practice:**  
Always call `Shutdown(ctx)` for OpenTelemetry injectors (`Tracer`, `Meter`, `TelemetryLogger`) at the end of your `main()`  
to flush all spans, metrics, and logs before exiting the program.

---

## 🧩 How to Create Your Own Injector

You can easily create your own injectors to provide new services (e.g., a database client, a feature flag SDK, or a custom REST client).

🧩 Injector contract:
Each injector must implement a single method —

```go
Inject(stageName string) any
```

It returns a pointer to the configured module or client (e.g. *zap.Logger, *redis.Client, etc.).
pipeyard automatically detects it and injects the returned instance into any stage requesting that type.

The essence of an injector is that it carries its configuration inside its own fields, making each instance self-contained and reusable across stages.

Example:

```go
type RedisInjector struct {
  Addr string
}

func (r *RedisInjector) Inject(stageName string) *redis.Client {
  fmt.Printf("Connecting Redis for stage %s at %s...", stageName, r.Addr)
  return redis.NewClient(&redis.Options{Addr: r.Addr})
}
```

Then simply register it in your pipeline:

```go
pipe := pipeline_provider.New(pipeline_provider.PipelineConfig{
  Injectors: []any{
    &RedisInjector{Addr: "localhost:6379"},
  },
})
```

Any stage that declares a `*redis.Client` parameter will automatically receive it at runtime.

---

## 🧭 Dependency Injection under the hood 

pipeyard automatically detects what each stage needs based on its `Execute()` parameters.

```go
func (s *TestStage) Execute(
  ctx context.Context,
  cmd *cobra.Command,
  logger *zap.Logger,
  shell *shell_executor.ShellExecutor,
  meter metric.Meter,
  tracer *trace.Tracer,
) error {
  ...
}
```

When you run `go run ./ci test`, pipeyard:
- inspects the `Execute` signature via reflection,
- matches each parameter with a registered injector,
- injects the correct instance.

No globals. No configuration duplication.

---

# 📊 Observability baked-in (explicit job instrumentation)

pipeyard doesn’t automatically instrument your stages — instead, it provides a **clean, reusable helper** called `JobInstrumentation` for **manual but standardized observability**.

Each stage can declare a tracing, metrics, and logging context using:

```go
ji := stage_layers.StartNewJobInstrumentation(ctx, "build", tracer, meter, logger)
defer ji.End(err)
```

This records:
- **Traces** (OpenTelemetry spans)
- **Metrics** (`job_total_calls`, `job_duration_seconds`)
- **Structured logs** (JSON logs with stage name, status, and trace ID)

Everything is correlated by `stage_name` and `trace_id`.

---

## 🧱 Example: instrumenting a stage

```go
func (s *BuildStage) Execute(ctx context.Context, cmd *cobra.Command, logger *zap.Logger, tracer *trace.Tracer, meter metric.Meter) error {
  ji := stage_layers.StartNewJobInstrumentation(ctx, "build", tracer, meter, logger)
  defer func() { ji.End(err) }()

  logger.Info("Building module...")
  err := exec.CommandContext(ctx, "go", "build", "./...").Run()
  return err
}
```

---

## 🪵 Example log output

```json
{"level":"info","msg":"🚀 Stage démarré","stage":"build"}
{"level":"info","msg":"Building module...","stage":"build"}
{"level":"info","msg":"✅ Stage terminé","stage":"build","duration_sec":3.2,"trace_id":"b6a12a2f2e3f3c9d"}
```

## 💡 Why it matters

This pattern:
- Keeps **full control** in your hands — you decide *where* to instrument.
- Ensures **consistent telemetry** across all stages.
- Standardizes how logs, metrics, and traces are emitted.
- Makes each stage observable, debuggable, and measurable.

> 🧠 You can still use the raw `tracer`, `logger`, and `meter` directly if you need more fine-grained control or want to add custom spans and metrics beyond the default stage-level instrumentation.

## 🧱 Self-powered pipeline

pipeyard uses *itself* to test, build, and release its library:
```bash
go run ./ci test
go run ./ci build
go run ./ci release v1.0.0
```

You can **extend pipeyard** with your own stages or injectors  
and still use the same CI/CD philosophy across all your projects.

---

## ✅ Summary

✔️ Centralized orchestration of CI/CD pipelines  
✔️ Declarative Go DSL  
✔️ Automatic dependency injection  
✔️ Built-in OpenTelemetry observability  
✔️ Safe shutdown of all telemetry providers  
✔️ Portable across GitHub Actions, GitLab CI, Jenkins, Dagger  
✔️ Extensible with custom injectors  

---

# 🌀 Quickstart

pipeyard is extremely simple to get started with — no boilerplate, no YAML.

---

## ⚙️ 1. Create your project

```bash
mkdir my-pipeline && cd my-pipeline
go mod init my-pipeline
go get github.com/rridane/golang-pipeyard
```

---

## 🧩 2. Create a simple pipeline

In `main.go`:

```go
package main

import (
    "context"
    "github.com/rridane/golang-pipeyard"
    "github.com/rridane/golang-pipeyard/injectors"
    "go.uber.org/zap/zapcore"
    "os"
)

type HelloStage struct{}

func (s *HelloStage) Configure(cmd *cobra.Command) {
    cmd.Use = "hello"
    cmd.Short = "Say hello from your first pipeyard stage"
}

func (s *HelloStage) Execute(ctx context.Context, cmd *cobra.Command, logger *zap.Logger) error {
    logger.Info("👋 Hello from pipeyard!")
    return nil
}

func main() {
    tracer := injectors.TracerInjector{
        ServiceName: "my-pipeline",
        OtelExporterOtlpEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
    }

    logger := injectors.TelemetryLoggerInjector{
        ServiceName: "my-pipeline",
        EnableOtelExport: true,
        OtelExporterOtlpEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
        LogLevel: zapcore.InfoLevel,
    }

    pipe := pipeline_provider.New(pipeline_provider.PipelineConfig{
        Name: "my_pipeline",
        ShortDescription: "A simple pipeyard demo",
        Version: "0.1.0",
        Injectors: []any{&tracer, &logger},
    })

    pipe.RegisterStage("hello", &HelloStage{})
    pipe.Run()
}
```

---

## 📡 3. Set up telemetry (optional)

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
```

You can use the [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) locally to view traces, logs, and metrics.

---

## 🚀 4. Run your pipeline

```bash
go run . hello
```

Output:

```
{"level":"info","msg":"👋 Hello from pipeyard!","stage":"hello"}
✅ Stage terminé en 0.004s
```

---

Enjoy a fully observable, programmable CI pipeline — all in Go. 🚀
