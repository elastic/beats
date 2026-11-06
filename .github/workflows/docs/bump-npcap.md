# bump-npcap

Documents the bump-npcap GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/bump-npcap.yml`](../bump-npcap.yml) |
| Trigger | `workflow_dispatch`, `schedule` (scheduled) |

## Purpose

Keeps the [Npcap](https://npcap.com/) OEM installer version used by Packetbeat up to date. The workflow runs updatecli to detect the npcap version published in the `elastic/golang-crossbuild` repository, verifies that a matching crossbuild Docker image is available, and opens a pull request updating `x-pack/packetbeat/scripts/mage/pcap.go`, the npcap installer LICENSE file, and a changelog fragment. This keeps Packetbeat's Windows packet-capture dependency aligned with the crossbuild toolchain without manual intervention.

## When it runs

- `workflow_dispatch` — triggered manually from the Actions UI.
- `schedule` — Runs on a schedule. See the [workflow configuration](../bump-npcap.yml) for current timing.

## Execution flow

| Job | Runner | Depends on | Configured work |
| --- | --- | --- | --- |
| bump | `ubuntu-latest` | None declared | Logs in to the Elastic Docker registry; runs updatecli `pipeline apply` with `.github/workflows/updatecli.d/bump-npcap.yml` and `.github/workflows/updatecli.d/values.d/scm.yml`; on failure sends a Slack notification to `#ingest-notifications` |

The updatecli pipeline it invokes:

1. Reads the Go version from `.go-version`.
2. Looks up `NPCAP_VERSION` from `Makefile.common` in the matching branch of `elastic/golang-crossbuild` (falls back to `main` if the branch does not exist).
3. Constructs the expected crossbuild image tag (`<go-version>-npcap-<npcap-version>-debian9`).
4. Skips the update if `NpcapVersion` in `x-pack/packetbeat/scripts/mage/pcap.go` already matches.
5. Checks that `docker.elastic.co/observability-ci/golang-crossbuild:<tag>` is published.
6. If a new version is confirmed available, opens a pull request updating:
   - `x-pack/packetbeat/scripts/mage/pcap.go` (`NpcapVersion` constant)
   - `x-pack/packetbeat/npcap/installer/LICENSE` (version line)
   - A new changelog fragment under `changelog/fragments/`

The pull request carries the labels `automation`, `dependencies`, and `backport-active-all`.

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Outputs | Pull request opened by updatecli when a new npcap version is detected; no artifact upload or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read"}`; bump: `{"contents": "write", "pull-requests": "write"}` |
| Secrets | `ELASTIC_DOCKER_REGISTRY`, `ELASTIC_DOCKER_USERNAME`, `ELASTIC_DOCKER_PASSWORD` — Docker registry login; `GITHUB_TOKEN` — updatecli SCM and PR creation; `SLACK_BOT_TOKEN` — failure notification |

## Operational notes

On updatecli failure the job posts a message to the `#ingest-notifications` Slack channel with a link to the failed run.

## Sources

- Workflow: [`.github/workflows/bump-npcap.yml`](../bump-npcap.yml)
- Local dependency: [`.github/workflows/updatecli.d/bump-npcap.yml`](../updatecli.d/bump-npcap.yml)
- Local dependency: [`.github/workflows/updatecli.d/values.d/scm.yml`](../updatecli.d/values.d/scm.yml)
