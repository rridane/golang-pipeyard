# Pipeyard

A Go framework for writing CI/CD stages as ordinary functions, shipped as one
binary per organisation.

Your projects stop carrying their pipeline logic. They download an executable
and call it.

```yaml
# .gitlab-ci.yml, in every project
test:
  script: corp-ci test --timeout 5m
build:
  script: corp-ci build --version $CI_COMMIT_TAG --push
```

Status: **v0, API in motion.** It builds, it pushes images that run, and a
failing stage fails the job. The rough edges are listed at the bottom.

---

## The problem it addresses

A pipeline that lives in YAML, in forty repositories, has no tests, no types,
no refactoring, and no version. When the base image changes, someone opens
forty merge requests. When a `script:` block grows past ten lines, nobody
reviews it any more.

Pipeyard moves that logic into Go. A stage becomes a function you can read,
test and version. The CI engine keeps what it is good at — sequencing, matrix
builds, artifacts between jobs — and calls your binary for the work itself.

Pipeyard is not an orchestrator. There is no DAG, no conditional branching.
One invocation, one stage, one exit code.

---

## A stage

```go
type TestStage struct{}

func (s *TestStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("test")
	spec.SetDescription("Run the test suite")
	spec.AddFlag("pkg", ru.StringFlag, "packages to test", "./...", false)
	spec.AddFlag("timeout", ru.DurationFlag, "per-test timeout", "5m", false)
	return spec
}

func (s *TestStage) Execute(
	ctx context.Context,
	args ru.ParsedArgs,
	sh *mod.ShellExecutor,
	log *zap.Logger,
) error {
	log.Info("running tests", zap.String("pkg", args.String("pkg")))

	return sh.Run(ctx, fmt.Sprintf("go test -timeout %s %s",
		args.Duration("timeout"), args.String("pkg")))
}
```

Two methods. `Configure` declares the command and its flags. `Execute` does the
work and returns an error — if it does, the process exits non-zero and the CI
job fails.

`Execute` is deliberately absent from the `Stage` interface. That is what lets
each stage ask for exactly the dependencies it needs, in its own signature.
Those parameters are filled in by injectors, matched by type.

---

## Wiring the binary

```go
package main

func main() {
	pipe := pi.NewPipeline(pi.PipelineConfig{
		Name:             "corp-ci",
		Version:          "1.4.0",
		ShortDescription: "CI pipeline for the payments team",
		Injectors: []any{
			&inj.TelemetryLoggerInjector{ServiceName: "corp-ci"},
			&inj.ShellExecutorInjector{Timeout: 10 * time.Minute},
			&inj.ContainerBuilderInjector{},
		},
	})

	pipe.RegisterStage(&TestStage{}).
		RegisterStage(&BuildStage{}).
		RegisterStage(&DeployStage{})

	pipe.Run(context.Background())
}
```

```
$ corp-ci --help
CI pipeline for the payments team

Usage:
  corp-ci [command]

Available Commands:
  build
  deploy
  test        Run the test suite

$ corp-ci test --pkg ./internal/... --timeout 2m
```

A stage whose `Execute` cannot be dispatched — wrong name, no return value, a
return that is not an `error` — makes the binary refuse to start, with the
reason. The compiler cannot catch those, so registration does.

---

## Building images

The container builder takes a Dockerfile and hands it to BuildKit. It does not
reimplement the format: multi-stage builds, cache mounts, secret mounts, build
args and wildcards in `COPY` all work because they are plain Dockerfile.

```go
func (s *BuildStage) Execute(
	ctx context.Context,
	args ru.ParsedArgs,
	cb *mod.ContainerBuilder,
) error {
	recipe := mod.Dockerfile{
		Args: map[string]any{"JDK": "21"},
		Template: `
			FROM maven:3.9-eclipse-temurin-{{.JDK}} AS builder
			WORKDIR /src
			COPY pom.xml .
			RUN --mount=type=cache,target=/root/.m2 mvn -B dependency:go-offline
			COPY src/ ./src/
			RUN --mount=type=cache,target=/root/.m2 mvn -B verify

			FROM eclipse-temurin:{{.JDK}}-jre-alpine AS runtime
			RUN addgroup -S svc && adduser -S svc -G svc
			WORKDIR /app
			COPY --from=builder /src/target/*.jar /app/app.jar
			USER svc
			EXPOSE 8080
			ENTRYPOINT ["java","-jar","/app/app.jar"]
		`,
	}

	image, err := cb.Build(ctx, mod.BuildRequest{
		Dockerfile: recipe,
		ContextDir: ".",
		Target:     "runtime",
	})
	if err != nil {
		return err
	}

	return image.Push(ctx, "harbor.corp/payments/service:"+args.String("version"))
}
```

`Template` is a Go string, so the recipe can be computed — and tested without
BuildKit:

```go
func TestRecipeRunsAsNonRoot(t *testing.T) {
	if !strings.Contains(paymentsRecipe().Render(), "USER svc") {
		t.Fatal("the runtime image would run as root")
	}
}
```

`Args` is optional. Leave it nil and `Template` is used verbatim, braces
included.

Once built, an image can go several ways:

```go
image.Push(ctx, ref)                        // to a registry
image.PushInsecure(ctx, ref)                // registry without TLS
image.SaveTar(ctx, ref, "./out/image.tar")  // archive for docker load
image.ExtractTo(ctx, "/src/target/*.jar", "./dist")
report, err := image.ReadFile(ctx, "/src/coverage.xml")
```

`ReadFile` is how a stage reads a value its build computed — a coverage report,
a version, a manifest — without going through the host disk.

The BuildKit endpoint comes from the `Endpoint` field, then `$BUILDKIT_HOST`,
then `tcp://127.0.0.1:1234`.

---

## Injectors

An injector is a struct with one method, `Inject(stageName string) *T`. The
executor matches each `Execute` parameter against the return type of every
registered injector.

| Injector | Provides |
|---|---|
| `TelemetryLoggerInjector` | `*zap.Logger`, optional OTLP export |
| `TracerInjector` | `trace.Tracer` |
| `MeterInjector` | `metric.Meter` |
| `ShellExecutorInjector` | `*mod.ShellExecutor` |
| `ContainerBuilderInjector` | `*mod.ContainerBuilder` |
| `FileSystemProcessorInjector` | `*mod.FileSystemProcessor` |
| `AssetFinderInjector` | `*mod.AssetFinder` |
| `MailerInjector` | `*mod.MailSender` |
| `K8sInjector` | `*kubernetes.Clientset` |
| `HelmInjector` | `*action.Configuration` |
| `TerraformInjector` | `*tfexec.Terraform` |
| `UnleashInjector` | `*unleash.Client` |
| `GitHubReleaseInjector` | `*mod.GithubReleaseSDK` |
| `GitlabReleaseInjector` | `*mod.GitlabRelease` |
| `DaggerInjector` | `*dagger.Client` |

Writing your own takes a struct and a method:

```go
type VaultInjector struct {
	Address string
	Token   string
}

func (v *VaultInjector) Inject(stageName string) *vault.Client {
	client, err := vault.New(v.Address, v.Token)
	if err != nil {
		return nil
	}
	return client
}
```

Add it to `Injectors`, and any stage can ask for a `*vault.Client`.

The `stageName` argument is there so a dependency can be named after the stage
that uses it: a tracer per stage, a logger with the right field, a cache key.

---

## Shell commands

```go
// streams to stdout, fails on a non-zero exit or on Timeout
err := sh.Run(ctx, "helm upgrade --install svc ./chart --wait")

// returns what the command wrote, for a value you need in Go
out, err := sh.Output(ctx, "git describe --tags --abbrev=0")
```

Scripts are interpreted by [mvdan.cc/sh](https://github.com/mvdan/sh), so no
system shell has to be present in the image.

---

## Observability

The three OTel signals are wired, each behind its own injector, each exporting
over OTLP gRPC when an endpoint is configured and falling back to a no-op
otherwise.

`x/lay/job_instrumentation.go` wraps a stage in a span, a counter and a
histogram, and logs the trace ID so a line in the CI output can be matched to a
trace.

`examples/_otel_local/` holds a compose file with Tempo, Prometheus, Grafana
and an OTel collector, for looking at it locally.

If the CI engine propagates `TRACEPARENT`, the separate invocations of your
binary end up in one distributed trace.

---

## Quickstart

```bash
mkdir corp-ci && cd corp-ci
go mod init github.com/corp/corp-ci
go get github.com/rridane/golang-pipeyard
```

Write a `main.go` like the one above, then:

```bash
go build -o corp-ci .
./corp-ci --help
```

Building container images needs a reachable BuildKit daemon:

```bash
docker run -d --name buildkit --privileged -p 1234:1234 \
  moby/buildkit --addr tcp://0.0.0.0:1234
```

Publish the binary as a release, and your projects fetch it:

```yaml
before_script:
  - curl -sSL -o /usr/local/bin/corp-ci https://.../releases/download/v1.4.0/corp-ci
  - chmod +x /usr/local/bin/corp-ci
```

---

## Pipeyard builds Pipeyard

`ci/` is this repository's own pipeline, written with the framework. The GitHub
workflow builds the module once as a smoke check, then calls the stages:

```bash
go run ./ci fmt            # gofmt, and names the files that need it
go run ./ci vet
go run ./ci test --timeout 5m
go run ./ci lint           # golangci-lint in a container, needs BuildKit
```

It is the shortest honest example of what a pipeline looks like here, and it is
the first thing to break when the framework does.

---

## Where it stands

What is covered, and verified against a real BuildKit daemon and registry:
multi-stage builds, cache and secret mounts, pushing an image that actually
starts, extracting artifacts, reading a file out of a build, a non-zero exit
when a stage or a shell command fails.

What is not there yet:

- **No git injector.** Deriving a version from tags, generating a changelog, or
  finding which modules of a monorepo changed has to go through `sh.Output`.
- **No companion services.** Integration tests that need a database stay with
  the CI engine's `services:`. BuildKit solves a hermetic graph; there is no
  shared network to attach a Postgres to.
- **One package of injectors.** Importing `x/inj` pulls in Kubernetes, Helm,
  Terraform, Dagger and BuildKit, which is most of a 98 MB binary. Splitting it
  into subpackages is the next structural change.
- **The release SDKs swallow their errors.** `x/mod/internal/github_releases_sdk`
  and its GitLab counterpart print failures and carry on. Do not rely on them
  yet.

---

## License

MIT. See [LICENSE](LICENSE).
