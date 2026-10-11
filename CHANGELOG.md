# Changelog

Versions follow semver. Until 1.0.0 the API can change between minor versions.
Breaking changes are listed first in each entry, because for a shared pipeline
binary they are what a project has to react to before moving its floating tag.

## 0.1.0 — 2026-10-11

First version meant to be used by someone other than its author.

### What it does

A stage is a Go function. It declares the dependencies it needs as parameters,
and the executor resolves them from the injectors a pipeline registers.

- **Stages** — one invocation, one stage, one exit code. The CI engine keeps
  the sequencing.
- **Injectors**, split by family across `x/inj/{otel,sys,container,k8s,iac,forge,notify,featureflag}`,
  so importing one does not link the others.
- **Layers** — reusable behaviour with its own typed context. `GoLintLayer` and
  `BuildAndPushLayer` ship.
- **A container builder** that hands a Dockerfile to BuildKit and gives back an
  artifact to push, save, extract from or read a file out of. Recipes are
  composable: one can be rendered into another.
- **A shell executor** that honours its context and its timeout, and reports
  what the command did.
- **Automatic instrumentation.** `PipelineConfig.Observer` wraps every stage in
  a span, a counter and a duration sample, with no code in the stages and no
  way for one to opt out. Leave it nil and nothing is recorded.
- **`ci/`** — this repository's own pipeline, which builds and releases
  Pipeyard: `build`, `fmt`, `vet`, `test`, `lint`, `release`, `publish`.

42 tests across 7 packages.

### What it needs from you

- **A collector that converts delta metrics.** Counters and histograms are
  exported with delta temporality, because a pipeline run is a process that
  starts, records one stage and exits — a cumulative counter would export the
  same `1` every run and `rate` over it would be zero. Add
  `deltatocumulative` to the metrics pipeline of your collector, which lives
  in the contrib distribution. Without it a Prometheus-compatible store
  rejects the metrics, and nothing in the pipeline tells you so.
- **Four environment variables**, so a record says where it came from:
  `PIPEYARD_PROJECT`, `PIPEYARD_BRANCH`, `PIPEYARD_COMMIT_SHA` and
  `PIPEYARD_COMMIT_TIME`. They are Pipeyard's own names; mapping your CI
  engine's variables onto them belongs in that engine's configuration.
- **A BuildKit daemon**, for a pipeline that builds images. `$BUILDKIT_HOST`,
  or `tcp://127.0.0.1:1234`.
