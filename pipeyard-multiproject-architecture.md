# 🧩 Multi-Project Architecture with Pipeyard  
> _Rethinking CI/CD as a Product_

## 🚀 Vision

Instead of treating CI pipelines as ephemeral YAML scripts hidden inside each repository,  
**Pipeyard** turns them into **first-class deliverables** — versioned, observable, and reusable across multiple projects.

Each CI becomes its own **product**:  
- **Versioned** (semantic versioning, releases, changelogs)  
- **Observable** (traces, logs, metrics via OpenTelemetry)  
- **Configurable** (feature-flagged behavior via Unleash)  
- **Portable** (binary executable used in any CI system)  

This approach allows your organization to **centralize and standardize** pipelines while keeping per-project customization through flags and configuration.

---

## 🏗️ Recommended Architecture

We recommend a **monorepo architecture** dedicated to your internal CI tooling:

```
pipeyard/
├── ci/
│   ├── java-pipeline/
│   │   ├── main.go
│   │   ├── build_stage.go
│   │   ├── test_stage.go
│   │   └── release_stage.go
│   ├── python-pipeline/
│   ├── flutter-pipeline/
│   └── shared/
│       ├── injectors/
│       ├── stage_layers/
│       └── modules/
└── workflows/
    ├── github/
    ├── gitlab/
    └── dagger/
```

Each folder under `ci/` represents a **fully self-contained pipeline**, versioned and released as a binary.

Example:
```bash
go build -o bin/java-ci ./ci/java-pipeline
go build -o bin/flutter-ci ./ci/flutter-pipeline
```

You can then **publish** these binaries in GitHub Releases or your internal artifact registry.

---

## 🔁 Usage in external projects

In each project’s CI (GitHub Actions, GitLab, Jenkins…),  
you simply **download and run the right CI binary**.

### Example (GitHub Actions)

```yaml
before_script:
  - curl -sL https://github.com/your-org/pipeyard/releases/download/v1.2.0/java-ci -o /usr/local/bin/java-ci
  - chmod +x /usr/local/bin/java-ci

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: 🧪 Run tests
        run: java-ci test
      - name: ⚙️ Build
        run: java-ci build
      - name: 🚀 Release
        run: java-ci release v1.2.0 --publish
```

Each pipeline binary can expose feature flags or toggles via **Unleash**:

```bash
UNLEASH_URL=https://unleash.company.com \
UNLEASH_API_TOKEN=$TOKEN \
java-ci build
```

Flags can enable experimental stages, additional linting, new deployment modes, etc.

---

## 🧠 CI as a Product

By packaging each pipeline as a **deliverable**, you gain:

| Benefit | Description |
|----------|--------------|
| **Consistency** | All projects follow the same validated stages and best practices |
| **Observability** | Unified telemetry (traces, logs, metrics) across every CI |
| **Reusability** | Pipelines are built once, used everywhere |
| **Feature-Flagging** | Progressive rollout of new CI features via Unleash |
| **Governance** | Versioned, testable, and reviewable pipelines |
| **Developer Experience** | Local runs: `./java-ci build` works exactly like the CI |

---

## 🧭 Recommended Practice

1. **Centralize your pipelines** in a `pipeyard` monorepo  
2. **Expose each CI as a binary** (per language or per product type)  
3. **Version them** (Git tags + GitHub Releases)  
4. **Distribute via artifact registry or GitHub releases**  
5. **Enable feature flags** via Unleash for incremental rollout  
6. **Instrument everything** with OpenTelemetry (Pipewire already does this)  
7. **Adopt semantic versioning** to align pipeline versions with your project releases  

---

## 🧩 Example lifecycle

| Step | Action |
|------|--------|
| **Develop** | Add or modify stages under `ci/java-pipeline` |
| **Test** | Run locally: `go run ./ci/java-pipeline build` |
| **Release** | Tag a new version: `v1.2.0` |
| **Distribute** | Publish binary `java-ci` in GitHub Releases |
| **Consume** | Each project uses that binary in CI via `curl` or package manager |

---

## 🌍 Result

You now have a **unified, versioned, observable pipeline ecosystem** —  
where every project’s CI is not a YAML script, but a **trusted software component**.
