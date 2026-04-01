## Repository Overview

Elastic Beats is a Go monorepo containing lightweight data shippers for the Elastic Stack. The module path is `github.com/elastic/beats/v7`. Go version: see `.go-version`.

## Build System

The primary build tool is **Mage** (Go-based). Makefile wraps mage for some targets. Each beat has its own `magefile.go`; shared build logic lives in `dev-tools/mage/`.

### Per-Beat Commands (run from the beat's directory, e.g. `cd filebeat`)

```bash
mage build              # Build the beat binary
mage unitTest           # Run Go unit tests
mage integTest          # Run integration tests (requires Docker)
mage goIntegTest        # Go integration tests only
mage pythonIntegTest    # Python integration tests only
mage update             # Regenerate fields, configs, dashboards, includes
mage fields             # Regenerate fields.yml and fields.go
mage config             # Regenerate config files
mage check              # Run checks (lint, headers, go mod)
mage crossBuild         # Cross-compile for all platforms
SNAPSHOT=true DEV=true PLATFORMS=$GOOS/$GOARCH PACKAGES=tar.gz mage package  # Build and package the beat
```

For `mage package`, set `PLATFORMS` to match the current OS/architecture. Check with `go env GOOS GOARCH` before running. Common values: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`. `PACKAGES` sets the format: `tar.gz`, `zip`, `rpm`, `deb`, `docker` (comma-separated).

### Running Tests

**NEVER run all unit or integration tests** (e.g. `mage unitTest` from root). They take too long and require too many dependencies. Always run tests scoped to the package you're working on:

```bash
# Run a single test or package tests
cd filebeat  # or any beat directory
go test -v -race -run TestName ./path/to/package/...
go test -v -race -run TestName -tags integration ./path/to/package/...  # integration tests

# Before running integration tests, build the system test binary if the beat defines it
mage buildSystemTestBinary
```

### Running a Beat

```bash
# Option 1: go run (from the beat directory)
go run . -e --strict.perms=false -c filebeat.yml

# Option 2: build then run
mage build
./filebeat -e --strict.perms=false -c filebeat.yml
```

Always pass `--strict.perms=false` during development — it disables strict config file permission checks.

### Pre-Push Check

Before pushing, run `mage check` from the beat directory — it's a required CI check and catches lint, header, and module issues early.

### Root-Level Commands

```bash
mage fmt                     # Format code + add license headers
make check                   # Full check suite (lint, headers, go mod, python)
```

### Linting

```bash
# golangci-lint v2 is configured via .golangci.yml
golangci-lint run ./...              # from any beat directory
mage linter:all                      # lint all
```

## Architecture

### Beats

| Beat | Description |
|------|-------------|
| `filebeat` | Lightweight agent that ships log files and journals |
| `metricbeat` | Collects metrics from systems and services |
| `heartbeat` | Monitors availability of services and endpoints, detecting uptime/downtime |
| `auditbeat` | Gathers audit data from systems to track security events, user activities, and compliance requirements |
| `packetbeat` | Analyzes network traffic by capturing and inspecting packets for application and network visibility |
| `winlogbeat` | Collects Windows Event logs for centralized monitoring and troubleshooting of Windows systems |
| `x-pack/osquerybeat` | Manages and queries Osquery endpoints (Elastic license) |
| `x-pack/dockerlogbeat` | Ships Docker container logs as a Docker log driver plugin (Elastic license) |
| `x-pack/otel` | OTel Collector components that wrap beats as receivers, processors, and exporters for the Elastic Distribution of OpenTelemetry (Elastic license) |

Each beat follows the pattern: `cmd/` (CLI entry), `beater/` (implements `beat.Beater` interface from `libbeat/beat/beat.go`), `module/` or `input/` (data collection).

### libbeat (shared framework)

All beats build on `libbeat/`, which provides:
- `beat/` — `Beater` interface (Run/Stop), `Beat` struct, `Pipeline`/`Client` for event publishing
- `outputs/` — Output plugins (Elasticsearch, Logstash, Kafka, Redis, etc.)
- `processors/` — Event processing pipeline
- `publisher/` — Internal publishing pipeline
- `cmd/` — Shared CLI infrastructure and beat instance bootstrapping
- `autodiscover/` — Dynamic service discovery (Docker, Kubernetes)
- `management/` — Elastic Agent management integration
- `statestore/` — Persistent state

### x-pack

`x-pack/` contains Elastic-licensed extensions for each beat and x-pack-only beats. Each `x-pack/{beat}` extends the corresponding OSS beat with additional modules, inputs, or features.

### Licensing Boundary

OSS code (Apache 2.0) **cannot** import from `x-pack/` or `elastic-agent-client`. This is enforced by `depguard` in `.golangci.yml`. X-pack code can import OSS code freely.

## Code Rules

- Logging: accept `*logp.Logger` as a parameter. For tests prefer `logptest`
- Use `github.com/stretchr/testify` in tests — always add a message explaining the failure
- Use `github.com/gofrs/uuid/v5`
- **Don't call `paths.Resolve` / `paths.InitPaths`** — use a per-beat `*paths.Path` instance
- **Don't use `math/rand`** — use `math/rand/v2`
- **goimports** uses local prefix `github.com/elastic` (imports grouped: stdlib, external, elastic)

## Code Formatting

```bash
mage fmt  # from root: runs goimports + python autopep8 + license headers
```

- Go: goimports with `github.com/elastic` local prefix
- Python: autopep8, max line length 120
- License headers: ASL2 for OSS code, Elastic for x-pack (applied by `go-licenser`)

## Changelog

PRs require a changelog fragment in `changelog/fragments/` (CI enforced unless `skip-changelog` label is applied). Create using `elastic-agent-changelog-tool`:

```bash
go run github.com/elastic/elastic-agent-changelog-tool@latest new --component <beat> --kind <kind>
```

Fragment format (`changelog/fragments/<timestamp>-<slug>.yaml`):
```yaml
kind: bug-fix          # bug-fix, enhancement, breaking-change, deprecation, known-issue
summary: Short description of the change
component: filebeat    # the affected beat/component
```

## Commits and PRs

When writing commit messages, explain WHAT changed and WHY — the rationale and motivation, not just a description of the diff.

When creating PRs, follow the template in `.github/PULL_REQUEST_TEMPLATE.md`