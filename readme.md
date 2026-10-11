# Pipeyard

A Go framework for writing CI/CD stages as ordinary functions, compiled into
one binary per organisation.

Your projects stop carrying their pipeline logic. Their CI configuration
becomes a few lines that call an executable, and then stops changing.

```yaml
# .gitlab-ci.yml, in every project
test:
  script: corp-ci test --timeout 5m
build:
  script: corp-ci build --version $CI_COMMIT_TAG --push
```

The pipeline itself moves to one repository, where it has a version, a
changelog, tests and a rollback. Every stage is instrumented without declaring
anything, so the pipeline reports its own DORA metrics. That is the part that is
hard to get any other way.

Status: **v0, API in motion.** It builds, it pushes images that run, and a
failing stage fails the job. The rough edges are listed at the bottom.

---

## The problem it addresses

A pipeline that lives in YAML, spread across forty repositories, has three
problems that compound.

**It cannot be tested.** The only way to find out whether a `script:` block
works is to push, wait for a runner, read the log, fix, push again. Every
attempt costs minutes and stays in the repository's history.

**It cannot be changed in one place.** When the base image moves, someone opens
forty merge requests. When a `script:` block grows past ten lines, nobody
reviews it any more.

**It has no version.** There is no way to say "put the CI back to what it was
on Tuesday" across forty projects.

Pipeyard moves that logic into Go. A stage becomes a function that runs the
same way on a laptop and on a runner, that `go test` can exercise, and that
ships as a versioned artifact. The CI engine keeps what it is good at —
sequencing, matrix builds, artifacts between jobs, triggers — and calls the
binary for the work itself.

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
			&otel.TelemetryLoggerInjector{ServiceName: "corp-ci"},
			&sys.ShellExecutorInjector{Timeout: 10 * time.Minute},
			&container.ContainerBuilderInjector{},
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

## Distributing the pipeline

The binary is built and published once, by the repository that owns the
pipeline. Consumer projects never build it.

**Publish on release, run an image everywhere else.** A tag on the pipeline
repository builds the binary, bakes it into an image, and pushes both. Every
other project runs that image:

```yaml
jobs:
  build:
    container: docker.io/corp/ci:1
    steps:
      - uses: actions/checkout@v4
      - run: corp-ci test --timeout 5m
```

The CI logic now goes through the same cycle as application code: a release
candidate you can run locally against a real project checkout, a tag, a
changelog, and a rollback.

Two consequences are worth stating plainly.

**Where the version lives decides whether you solved anything.** Fetching the
binary with `curl .../releases/download/v1.4.0/corp-ci` puts the version back
into forty YAML files: you removed the duplicated logic and kept the duplicated
version. A floating tag (`ci:1`, `ci:stable`) keeps it in one place — you move
the tag, the next run picks it up, no project changes.

**Centralising concentrates the blast radius.** Forty drifted YAML files are
forty small problems that fail independently. One shared binary is one bad
release breaking forty pipelines at once. The mitigation is release discipline,
not a feature: a canary tag on two projects, a stable tag on the rest, pinning
available to a project that needs to hold back, and a changelog that says what
changed in the logic. Put that in place before asking an organisation to
centralise.

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

| Package | Injector | Provides |
|---|---|---|
| `x/inj/otel` | `TelemetryLoggerInjector` | `*zap.Logger`, optional OTLP export |
| | `TracerInjector` | `trace.Tracer` |
| | `MeterInjector` | `metric.Meter` |
| `x/inj/sys` | `ShellExecutorInjector` | `*mod.ShellExecutor` |
| | `FileSystemProcessorInjector` | `*mod.FileSystemProcessor` |
| | `AssetFinderInjector` | `*mod.AssetFinder` |
| `x/inj/container` | `ContainerBuilderInjector` | `*mod.ContainerBuilder` |
| `x/inj/k8s` | `K8sInjector` | `*kubernetes.Clientset` |
| | `HelmInjector` | `*action.Configuration` |
| `x/inj/iac` | `TerraformInjector` | `*tfexec.Terraform` |
| `x/inj/forge` | `GitHubReleaseInjector` | `*mod.GithubReleaseSDK` |
| | `GitlabReleaseInjector` | `*mod.GitlabRelease` |
| | `GitlabApiInjector` | `*mod.GitlabApi` — issues, tags and releases of a project |
| `x/inj/notify` | `MailerInjector` | `*mod.MailSender` |
| | `TeamsInjector` | `*mod.TeamsSDK` |
| `x/inj/featureflag` | `UnleashInjector` | `*unleash.Client` |

They are split by family on purpose: importing a package links everything it
imports, so one package of injectors would make every pipeline pay for Helm and
Kubernetes. A pipeline that builds and pushes an image weighs 26 MB; one that
also deploys with Helm weighs 88 MB, because it genuinely uses Helm.

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

One injector is built in: a stage that declares `pi.Ctx` receives the pipeline's
working directory and its mount directory, the latter from `$PIPEYARD_MOUNT_DIR`
with `./mounts` as the default.

```go
func (s *DeployStage) Execute(ctx context.Context, args ru.ParsedArgs, pyctx pi.Ctx) error {
	manifests := filepath.Join(pyctx.MountDir, "k8s")
	...
}
```

---

## Layers

A layer is a reusable piece of a pipeline: a struct, a name, and a `Run` that
takes a context value. Where an injector provides a *client*, a layer provides a
*behaviour* — something several stages do the same way.

```go
type Layer[C any] interface {
	Name() string
	Run(ctx context.Context, pipeyard pi.Ctx, c C) error
}
```

`C` is the layer's own context type, so each layer declares exactly what it
needs and the compiler checks the call site:

```go
func (s *LintStage) Execute(ctx context.Context, args ru.ParsedArgs,
	cb *mod.ContainerBuilder, log *zap.Logger) error {

	layer := lay.GoLintLayer{CBuilder: cb, Logger: log}

	return layer.Run(ctx, pi.Ctx{CurrentDir: "."}, lay.GoLintLayerContext{
		LintImage: "golangci/golangci-lint:v1.62-alpine",
	})
}
```

Two layers ship today: `GoLintLayer` runs golangci-lint in a container with the
Go caches mounted, and `BuildAndPushLayer` builds an image from a Dockerfile on
disk and delivers it to a registry or a tarball.

A layer is a plain struct, so it belongs in the repository that owns the
pipeline, next to the stages that use it. That is where the sharing between
projects actually happens: a stage is specific, a layer is the part worth
publishing.

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

## What the binary depends on

Every `sh.Run` is a dependency on whatever happens to be installed on the
runner. Every injector is not:

| | |
|---|---|
| `HelmInjector` | the Helm Go library — no `helm` binary |
| `K8sInjector` | client-go — no `kubectl` |
| `ContainerBuilder` | gRPC to BuildKit — no Docker CLI |
| `TerraformInjector` | `exec.LookPath("terraform")` — **needs the binary** |
| `ShellExecutor` | runs whatever you name |

So the injectors are not dependency-injection sugar: they are what makes the
binary self-sufficient. A pipeline written with injectors alone is one static
executable with nothing to install around it, and a plain download is enough to
ship it. A pipeline that leans on `sh.Run` needs an image that carries its
tools.

Worth knowing before choosing how to distribute: the shell is the escape hatch,
and it costs hermeticity. `TerraformInjector` is the one injector that breaks
it, because `terraform-exec` drives the real binary.

---

## Observability

Give the pipeline an observer and every stage is instrumented. No stage declares
anything, and no stage can opt out — which is the point: an unrecorded
deployment reads as a lower deployment count, not as an error.

```go
tracer := &otel.TracerInjector{ServiceName: "corp-ci", OtelExporterOtlpEndpoint: endpoint}
meter  := &otel.MeterInjector{ServiceName: "corp-ci", OtelExporterOtlpEndpoint: endpoint}
logger := &otel.TelemetryLoggerInjector{ServiceName: "corp-ci", EnableOtelExport: true,
	OtelExporterOtlpEndpoint: endpoint}

pi.NewPipeline(pi.PipelineConfig{
	Observer: &lay.StageObserver{
		Tracer: tracer.Inject("corp-ci"),
		Meter:  meter.Inject("corp-ci"),
		Logger: logger.Inject("observer"),
		Flags:  []string{"env"},
		Shutdown: []func(context.Context){
			tracer.Shutdown, meter.Shutdown, logger.Shutdown,
		},
	},
	...
})
```

Leave `Observer` nil and nothing is recorded. That is the on/off switch: one
field, not a flag you could set without wiring an exporter behind it.

`Shutdown` is not optional in a short-lived command. A batching exporter keeps
what it collected in memory, and `Run` is the only place that knows the process
is about to exit — `os.Exit` skips deferred calls. Without it the pipeline
produces telemetry and throws it away.

### What is recorded

| | |
|---|---|
| span | `{stage}.execute`, with `stage`, `status`, `duration_sec`, `commit_sha`, `commit_time` |
| counter | `stage_runs_total{stage, status}` |
| histogram | `stage_duration_seconds{stage, status}` |
| log | start and outcome, with the trace id when a tracer is wired |

`Flags` adds the value of named stage flags to both the metrics and the span, so
`deploy --env staging` and `deploy --env production` are separate series. Keep
that list low-cardinality: a version or an image reference would create one
metric series per release.

The commit identity is deliberately on the span only, never on the metrics — a
commit timestamp as a Prometheus label is one series per commit.

A stage that panics is recorded as a failure before the panic propagates. An
unrecorded crash would be the one failure the failure rate must not miss.

### DORA metrics

Which of the four you get depends on the stages you name, not on the framework.

A pipeline with a single `deploy` stage gives you deployment frequency, and a
change failure rate that is really the job's failure rate: a deploy that exits
zero and breaks production counts as a success.

Split that stage in three and the picture changes.

```go
pipe.RegisterStage(&DeployStage{}).    // helm upgrade --wait
	RegisterStage(&VerifyStage{}).     // the smoke test, on its own
	RegisterStage(&RollbackStage{})    // helm rollback
```

```yaml
deploy:
  script: corp-ci deploy --env production --version $CI_COMMIT_TAG
verify:
  script: corp-ci verify --env production
rollback:
  script: corp-ci rollback --env production
  when: on_failure
```

| | |
|---|---|
| **Deployment frequency** | `sum(rate(stage_runs_total{stage="deploy",status="success",flag_env="production"}[7d]))` |
| **Change failure rate** | `stage_runs_total{stage="verify",status="error"} / stage_runs_total{stage="verify"}` — a failed verify is a degradation in production, which is the DORA definition and not a proxy |
| **Lead time for changes** | from the traces: `commit_time` on the span, against the end of a successful `deploy`. GitLab exposes `CI_COMMIT_TIMESTAMP`; on GitHub Actions set `PIPEYARD_COMMIT_TIME` yourself |
| **Time to restore service** | the gap between a failed `verify` and the next successful `deploy` or `rollback`. Not an incident MTTR — the restore time as the pipeline sees it |

One metric for free, four for a pipeline that names its steps. And none of the
stages declares any of it: naming a stage is what produces its series.

Note where the conditional lives: `when: on_failure` is the CI engine's job, and
it stays there. Pipeyard only has to make `rollback` a stage of its own.

`examples/_otel_local/` holds a compose file with Tempo, Prometheus, Grafana and
an OTel collector, for looking at all of it locally.

If the CI engine propagates `TRACEPARENT`, the separate invocations of your
binary end up in one distributed trace — build, test and deploy under one root.

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

`ci/` is this repository's own pipeline, written with the framework: one file
per stage, one file for registration. The GitHub workflow compiles it once and
calls it:

```yaml
- name: Build
  run: |
    go build ./...
    go build -o bin/pipeyard-ci ./ci

- name: Format
  run: bin/pipeyard-ci fmt
- name: Vet
  run: bin/pipeyard-ci vet
- name: Test
  run: bin/pipeyard-ci test --timeout 5m
```

That workflow is now nearly all of the YAML this project will ever need. Adding
a stage, changing a flag or fixing a check happens in Go, under test, and runs
the same way on a laptop:

```bash
go build -o bin/pipeyard-ci ./ci
bin/pipeyard-ci fmt
bin/pipeyard-ci test --pkg ./pi/ --race=false
bin/pipeyard-ci lint                        # golangci-lint in a container
bin/pipeyard-ci release --version 0.1.0     # builds its own distribution image
```

No push, no failed job, no second push.

**This repository is the one exception to the section above: it compiles the
binary in its own CI instead of running a published image.** A pipeline change
has to be validated by the pipeline it changes, the way a compiler is built
with the previous compiler and then runs its own tests. Do not copy this
workflow into a consumer project — copy the one under *Distributing the
pipeline*.

---

## Releasing

The `release` stage builds the distribution image with Pipeyard's own container
builder, extracts the binary, and pushes two tags: the exact version and its
major, so a consumer can pin `0.1.0` or follow `0`. Pipeyard publishes its own
to `rridane/pipeyard`.

```bash
bin/pipeyard-ci release --version 0.1.0 --push
```

Pushing a `v*` tag runs that stage in CI. Nothing else produces the binary.

```bash
git tag -a v0.1.0 -m "first usable version"
git push origin v0.1.0
```

Breaking changes are listed first in each [CHANGELOG](CHANGELOG.md) entry,
because for a shared pipeline binary they are the entries that matter: they are
what a project has to react to before moving its floating tag.

---

## Where it stands

What is covered, and verified against a real BuildKit daemon and registry:
multi-stage builds, cache and secret mounts, pushing an image that actually
starts, extracting artifacts, reading a file out of a build, a non-zero exit
when a stage or a shell command fails.

What is not there yet:

- **No published image yet.** The release stage and the workflow that calls it
  exist and have been run against a local registry, but no tag has been pushed,
  so `rridane/pipeyard` is still empty.
- **The release SDKs swallow their errors.** `github_releases_sdk` and its
  GitLab counterpart return themselves for chaining, print failures and carry
  on, and `gitlab_release_sdk` still calls `log.Fatal` on a validation error.
  Do not rely on them yet.
- **No git injector.** Deriving a version from tags, or finding which modules of
  a monorepo changed, has to go through `sh.Output` —
  which also costs the binary its hermeticity, since it then needs git on the
  runner.
- **No companion services.** Integration tests that need a database stay with
  the CI engine's `services:`. BuildKit solves a hermetic graph; there is no
  shared network to attach a Postgres to.
- **`x/mod` is still one package.** Asking it for a `ShellExecutor` links
  BuildKit, so every pipeline starts at 26 MB. That is the price of the
  container builder, which is the point of the tool, but a pipeline that never
  builds an image pays it too.

---

## License

MIT. See [LICENSE](LICENSE).
