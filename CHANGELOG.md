# Changelog

Versions follow semver. Until 1.0.0 the API can change between minor versions.
Breaking changes are listed first in each entry, because for a shared pipeline
binary they are what a project has to react to before moving its floating tag.

## 0.1.2 — 2026-10-11

### Changed

- **Every log line locates itself.** A body is now prefixed `[stage
  branch@sha]`. A log viewer shows the body and collapses the attributes, and
  lines from several runs and several jobs interleave there, so one context
  line per run — what 0.1.1 shipped — is lost among the others. An alert
  matching `vet started` has to match `started` instead: the stage moved from
  the message into the prefix, and the observer's lines are now `started`,
  `finished` and `failed`.
- The commit title stays on the opening line only. It is prose of variable
  length, and repeating it would crowd out the message it precedes.

### Added

- `x/inj/otel` had no tests and now has three, over the log body and the run
  scope. 54 tests across 9 packages.

## 0.1.1 — 2026-10-11

Telemetry, from what it records to how it reads. Nothing here changes a
signature; everything here changes a query.

### Changed

- **The log messages name their stage**, and the opening line of each run
  carries the branch, the short commit and the commit title. `stage started`
  said nothing when six stages wrote it in turn.
- **`branch` is a label on the metrics**, next to `stage` and `status`, instead
  of only an attribute of the resource. A Prometheus store keeps resource
  attributes out of the series — Mimir puts them in `target_info` — so a branch
  on the resource could not be filtered on. `stage_runs_total{branch="main"}`
  answers now. Existing series gain a label, so an aggregation keeps working
  while an exact matcher on the whole label set does not.
- **The commit leaves the resource of the traces.** It rides the span, where
  lead time is computed from anyway. A `spanmetrics` connector derives metrics
  from traces and copies their resource along, which was turning the commit
  into one `target_info` series per commit — the cardinality the metrics
  already refused. A Tempo query on `resource.commit_sha` has to read the span
  attribute instead.

### Added

- **`PIPEYARD_COMMIT_TITLE`**, the fifth variable of the contract, so a line
  says what change it was testing. Only the first line of a commit message is
  kept; the body is not a log line. Read as a value, never interpolated into a
  shell.
- `pi.Identity.CommitTitle` and `pi.Identity.ShortSHA()`.
- `x/lay` had no tests and now has four, over the opening line and the metric
  labels.

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
