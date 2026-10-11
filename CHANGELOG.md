# Changelog

Versions follow semver. Until 1.0.0 the API can change between minor versions;
breaking changes are listed first in each entry.

## 0.1.0 — unreleased

First version meant to be used by someone other than its author.

### Breaking

- `ContainerBuilder` no longer exposes a fluent LLB API. `InitContainer`,
  `ExecShell`, `ExecScript`, `CopyFromHost`, `CopyToHost`, `Apply` and `Export`
  are gone, along with the twelve state mutators behind them. A build is now a
  Dockerfile handed to BuildKit: `cb.Build(ctx, BuildRequest{Dockerfile: ...})`
  returns an `Artifact` with `Push`, `SaveTar`, `ExtractTo` and `ReadFile`.
- Injectors are split by family. `x/inj` becomes `x/inj/otel`, `x/inj/sys`,
  `x/inj/container`, `x/inj/k8s`, `x/inj/iac`, `x/inj/forge`, `x/inj/notify`
  and `x/inj/featureflag`. Importing one package no longer links Helm,
  Kubernetes and Terraform: a build-and-push pipeline went from 89 MB to 26 MB.
- `DaggerInjector` removed. `ContainerBuilder` talks to BuildKit directly.
- `FilePathFlag` and `URLFlag` removed. Both were declared but rejected at
  parse time, so a stage using one could never run. A path and a URL are
  strings; validate them in the stage.
- `StageSpec.AddFlag` takes `any` for its default value instead of `string`.
  An `int` or `bool` default used to be silently dropped to the zero value.
- `ParsedArgs.Keys` returns a fourth slice, for duration flags.
- `PipelineConfig.Assets` removed. It was declared and never read; the field
  that works is `AssetFinderInjector.FS`.
- `TeamsSDK.SendNotification` returns `error`.
- `mod.ContainerSpec`, `mod.ContainerExportOptions` and the four
  `mod.ContainerModeExport*` constants are gone.
- `lay.JobInstrumentation` removed, along with `StartNewJobInstrumentation`.
  It was never called, it did not honour `Layer[C]`, and instrumentation a
  stage has to remember is instrumentation a stage forgets. Use
  `PipelineConfig.Observer`.
- `GoLintLayer.Run` honours `Layer[C]`: it takes `(ctx, pi.Ctx,
  GoLintLayerContext)` instead of `(ctx, *zap.Logger, *mod.ContainerBuilder)`.
  Both shipped layers now honour the contract.

### Fixed

- `TeamsSDK.SendNotification` returns an error instead of calling `log.Fatal`
  and `os.Exit`. An unreachable webhook used to kill the process, so a
  successful deploy followed by a failed notification produced a red job and
  a rollback for nothing.
- `OrderByFilter` and `PaginationFilter` satisfy `IssueFilter`. Their issue-side
  method was named `ToGitlabIssueOptions`, so it matched neither interface and
  the two filters could not be passed to anything.
- A stage that returns an error now fails the command. The executor read
  `Execute`'s return value nowhere, so every pipeline exited zero whatever
  happened.
- A failed container build returns an error. `solveGraph` returned `nil` on all
  of its error paths, and `Apply` had no way to report anything.
- A pushed image carries its configuration. The image exporter received no
  `containerimage.config`, so the result had no entrypoint, no user and no
  environment: it could not start.
- `ShellExecutor.Run` honours its context, applies `Timeout`, and reports parse
  errors and non-zero exits. It used to return `nil` unconditionally.
- A stage whose `Execute` cannot be dispatched is rejected at registration
  instead of silently succeeding. `Execute` is absent from the `Stage`
  interface by design, so the compiler cannot check it.
- An injector returning a value instead of a pointer no longer panics, and its
  dependency now resolves.
- `DurationFlag` works on both sides. Declaring one used to make the stage
  impossible to run.
- A build honours `.dockerignore`. Without it the whole directory was shipped
  as context, `.git` included, and the build failed on any unreadable file.
- `ShellExecutorInjector.Timeout` is exported. It was lowercase, so the
  documented setting could not be set.
- `Pipeline.Run` always exits non-zero on failure. A failed write to stderr
  made it return zero.

### Added

- `GitlabApiInjector` and `TeamsInjector`. Both clients existed and neither was
  reachable: no injector, and the types their signatures need were missing from
  the facade. `mod.GitlabTag`, `GitlabIssue`, `GitlabLink`, `GitlabReleaseInfo`,
  the two filter interfaces and the four concrete filters are exported now, as
  are `mod.TeamsNotification` and `TeamsNotificationButton`.
- `pi.Ctx` is injectable. A stage that declares it receives the working
  directory and the mount directory, the latter from `$PIPEYARD_MOUNT_DIR`.
  Nothing used to build one, so a caller had to write it by hand.
- `list` prints the stage names, one per line. `AddListCommand` existed and was
  never called.
- Automatic stage instrumentation. `PipelineConfig.Observer` wraps every stage
  in a span, a counter and a duration sample, with no code in the stages and no
  way for one to opt out. `lay.StageObserver` is the OpenTelemetry
  implementation; `pi` declares the interface and gains no dependency.
  Which DORA metrics come out of it depends on the stages a pipeline names:
  a single `deploy` gives deployment frequency, while `deploy`, `verify` and
  `rollback` as separate stages give all four, three of them exactly.
- `pi.Run` flushes the observer before exiting. A batching exporter in a
  short-lived command kept everything in memory, so the previous
  instrumentation produced telemetry and discarded it.
- A stage that panics is recorded as a failure before the panic propagates.
- `mod.Dockerfile{Template, Args}` — a build recipe, raw or rendered through
  `text/template`, composable by injecting one recipe into another.
- `Artifact.ReadFile` reads a file out of a build without touching the host
  disk, which is how a stage reads a value its build computed.
- `ShellExecutor.Output` returns what a command wrote on stdout.
- `ShellExecutor.Dir` sets the working directory.
- `ParsedArgs.Duration`, `DurationOk`, `RequireDuration`, `SetDuration`.
- `ContainerBuilder.Endpoint`, falling back to `$BUILDKIT_HOST` then
  `tcp://127.0.0.1:1234`. The endpoint used to be hardcoded in four places.
- `ci/` — this repository's own pipeline, one file per stage, including a
  `release` stage that builds and pushes the distribution image.
- A test suite that locks the fixes above: 20 tests across five packages.
- `LICENSE` (MIT), a CI workflow that actually runs, and a README whose
  examples compile.
