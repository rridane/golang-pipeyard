# Pipewire

> **A flexible, type-safe pipeline framework for Go — with dynamic dependency injection and native CLI support.**

---

## 🚀 Overview

**Pipewire** lets you define CLI pipelines as modular *stages*, each with its own dependencies, automatically injected at runtime.

It combines the power of:

* 🧩 **Cobra** (for CLI structure)
* 🧠 **Reflection** (for smart dependency injection)
* ⚙️ **OpenTelemetry**, **Dagger**, **Kubernetes**, **Terraform**, etc. (through ready-to-use injectors)

The goal: **write composable, observable pipelines — without boilerplate.**

---

## ✨ Quick Start

### 1️⃣ Install

```bash
go get github.com/rridane/pipewire
```

### 2️⃣ Minimal Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/rridane/pipewire"
    "github.com/spf13/cobra"
)

type HelloStage struct{}

func (s *HelloStage) Configure(cmd *cobra.Command) {
    cmd.Use = "hello <name>"
    cmd.Short = "Print a greeting"
    cmd.Args = cobra.ExactArgs(1)
}

func (s *HelloStage) Execute(ctx context.Context, cmd *cobra.Command) error {
    name := cmd.Flags().Args()[0]
    fmt.Printf("👋 Hello, %s!\n", name)
    return nil
}

func main() {
    pipeline := pipewire.New(pipewire.PipelineConfig{
        Name: "demo",
    })

    pipeline.RegisterStage("hello", &HelloStage{})
    pipeline.Run()
}
```

🧠 Run it:

```bash
go run . hello World
```

Output:

```bash
👋 Hello, World!
```

---

## 🧩 Advanced Example with Native Injectors

Pipewire provides optional **native injectors** to integrate directly with your tools.

```go
import (
    "context"
    "fmt"
    "dagger.io/dagger"
    "github.com/rridane/pipewire"
    "github.com/rridane/pipewire/injectors"
    "github.com/spf13/cobra"
    "go.uber.org/zap"
    v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
)

type DeployStage struct{}

func (s *DeployStage) Configure(cmd *cobra.Command) {
    cmd.Use = "deploy <app> <version>"
    cmd.Args = cobra.ExactArgs(2)
    cmd.Flags().StringP("namespace", "n", "default", "Kubernetes namespace")
}

func (s *DeployStage) Execute(
    ctx context.Context,
    cmd *cobra.Command,
    client *dagger.Client,
    logger *zap.Logger,
    k8s *kubernetes.Clientset,
) error {
    app := cmd.Flags().Args()[0]
    version := cmd.Flags().Args()[1]
    ns, _ := cmd.Flags().GetString("namespace")

    logger.Info(fmt.Sprintf("🚀 Deploying %s:%s in %s", app, version, ns))

    pods, err := k8s.CoreV1().Pods(ns).List(ctx, v1.ListOptions{})
    if err != nil {
        return err
    }

    fmt.Println(pods.String())
    return nil
}

func main() {
    pipe := pipewire.New(pipewire.PipelineConfig{
        Name:      "ci_pipeline",
        Injectors: []any{&injectors.DaggerInjector{}, &injectors.LoggerInjector{}, &injectors.K8sInjector{}},
    })

    pipe.RegisterStage("deploy", &DeployStage{})
    pipe.Run()
}
```

✅ Automatically injected:

* 🧱 Dagger client
* 📜 Zap logger
* ☸️ Kubernetes client-go

---

## 🧠 Create Your Own Injector

Adding a new dependency injector is as simple as implementing this interface:

```go
type DependencyInjector[T any] interface {
    Inject(stageName string) *T
}
```

Example — a **GitHub API client injector**:

```go
package injectors

import "github.com/google/go-github/v66/github"

type GitHubInjector struct{}

func (g *GitHubInjector) Inject(stageName string) *github.Client {
    return github.NewClient(nil)
}
```

Then register it in your pipeline:

```go
pipe := pipewire.New(pipewire.PipelineConfig{
    Name: "ci_pipeline",
    Injectors: []any{
        &injectors.LoggerInjector{},
        &injectors.GitHubInjector{},
    },
})
```

And just declare the dependency in your stage:

```go
func (s *ReleaseStage) Execute(ctx context.Context, cmd *cobra.Command, logger *zap.Logger, gh *github.Client) error {
    logger.Info("Creating GitHub release...")
    // Use gh.Repositories.CreateRelease(...)
    return nil
}
```

---

## 🧩 Built-in Injectors

| Injector            | Type                    | Description                           |
| ------------------- | ----------------------- | ------------------------------------- |
| `DaggerInjector`    | `*dagger.Client`        | Build & deploy containers via Dagger  |
| `LoggerInjector`    | `*zap.Logger`           | Structured logging with stage context |
| `TracerInjector`    | `*trace.Tracer`         | OpenTelemetry tracing support         |
| `K8sInjector`       | `*kubernetes.Clientset` | Kubernetes client-go access           |
| `TerraformInjector` | `*tfexec.Terraform`     | Run Terraform programmatically        |
| `HelmInjector`      | `*action.Configuration` | Use Helm programmatically             |

---

## ⚡ Debug & Validation

Pipewire validates all injectors at runtime:

```bash
❌ *MyBadInjector.Inject must have exactly one string parameter
❌ *MyOtherInjector.Inject must return a pointer
```

No surprises at execution — everything is type-checked and validated.

---

## 🧠 Philosophy

* **Explicit over implicit** → no hidden runtime magic
* **Composable** → plug your tools easily
* **Observability-first** → metrics & tracing included
* **Type-safe** → only pointers of known types are injected

Pipewire is designed for **Platform Engineers, DevOps, and Infra Developers** who want to build internal developer tools that scale.

---

## 📦 Example Folder Layout

```text
.
├── main.go
├── injectors/
│   ├── dagger.go
│   ├── logger.go
│   ├── k8s.go
│   ├── terraform.go
│   └── helm.go
└── stages/
    ├── deploy.go
    ├── build.go
    └── test.go
```

---

## 🧰 License

MIT License © 2025 [R. Ridane](https://github.com/rridane)

---

**Pipewire — Compose. Inject. Run.**
