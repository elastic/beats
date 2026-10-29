# Beats Tests Pipeline

Documents the Buildkite pipeline configuration in [`.buildkite/beats-tests-pipeline.yml`](../beats-tests-pipeline.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/beats-tests-pipeline.yml`](../beats-tests-pipeline.yml) |
| Trigger | Daily, by `beats-pipeline-scheduler`; also manually |

## Purpose

Runs the full test suite of every Beat on one commit, and publishes the result as the `beats-tests (daily)` GitHub commit status on every commit the build covered. Merges to `main` and release branches do not run these tests, so this status is how a merged commit shows whether the tests that included it passed.

## When it runs

`beats-pipeline-scheduler` triggers it daily at 04:00 UTC for each active branch listed in the artifacts API `branches.json`; see [`pipeline-scheduler.md`](pipeline-scheduler.md). It can also be triggered manually. Step-level conditions are described below.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| Triggering Build for Auditbeat | No step-level condition or agent declared | None declared | key `test-auditbeat`; triggers pipeline `auditbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for Heartbeat | No step-level condition or agent declared | None declared | key `test-heartbeat`; triggers pipeline `heartbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for Filebeat | No step-level condition or agent declared | None declared | key `test-filebeat`; triggers pipeline `filebeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for x-pack/filebeat | No step-level condition or agent declared | None declared | key `test-x-pack-filebeat`; triggers pipeline `beats-xpack-filebeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for x-pack/dockerlogbeat | No step-level condition or agent declared | None declared | key `test-x-pack-dockerlogbeat`; triggers pipeline `beats-xpack-dockerlogbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for Metricbeat | No step-level condition or agent declared | None declared | key `test-metricbeat`; triggers pipeline `beats-metricbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for x-pack/metricbeat | No step-level condition or agent declared | None declared | key `test-x-pack-metricbeat`; triggers pipeline `beats-xpack-metricbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for x-pack/osquerybeat | No step-level condition or agent declared | None declared | key `test-x-pack-osquerybeat`; triggers pipeline `beats-xpack-osquerybeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for x-pack/winlogbeat | No step-level condition or agent declared | None declared | key `test-x-pack-winlogbeat`; triggers pipeline `beats-xpack-winlogbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for Deploy/K8S | No step-level condition or agent declared | None declared | key `test-deploy-k8s`; triggers pipeline `deploy-k8s`; trigger build fields `["commit", "branch"]` |
| Triggering Build for Libbeat | No step-level condition or agent declared | None declared | key `test-libbeat`; triggers pipeline `beats-libbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for pkg/systemmetrics | No step-level condition or agent declared | None declared | key `test-pkg-systemmetrics`; triggers pipeline `beats-pkg-systemmetrics`; trigger build fields `["commit", "branch"]` |
| Triggering Build for x-pack/libbeat | No step-level condition or agent declared | None declared | key `test-x-pack-libbeat`; triggers pipeline `beats-xpack-libbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for x-pack/otel | No step-level condition or agent declared | None declared | key `test-x-pack-otel`; triggers pipeline `beats-xpack-otel`; trigger build fields `["commit", "branch"]` |
| Triggering Build for x-pack/auditbeat | No step-level condition or agent declared | None declared | key `test-x-pack-auditbeat`; triggers pipeline `beats-xpack-auditbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for x-pack/heartbeat | No step-level condition or agent declared | None declared | key `test-x-pack-heartbeat`; triggers pipeline `beats-xpack-heartbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for x-pack/packetbeat | No step-level condition or agent declared | None declared | key `test-x-pack-packetbeat`; triggers pipeline `beats-xpack-packetbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for Winlogbeat | No step-level condition or agent declared | None declared | key `test-winlogbeat`; triggers pipeline `beats-winlogbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for Packetbeat | No step-level condition or agent declared | None declared | key `test-packetbeat`; triggers pipeline `beats-packetbeat`; trigger build fields `["commit", "branch"]` |
| Wait | `continue_on_failure: true` | All trigger steps | Lets the next step run even when a triggered build fails |
| :github: Publish commit status | condition `build.branch == "main" \|\| build.branch =~ /^[0-9]+\.[0-9x]+$/`; agent image `docker.elastic.co/release-eng/wolfi-build-essential-release-eng:latest`; plugin `elastic/vault-github-token#v0.2.3` | Wait | runs [`.buildkite/scripts/daily-tests-commit-status.sh`](../scripts/daily-tests-commit-status.sh); `soft_fail: true`; manual retry allowed after it passed |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | `BUILDKITE_BRANCH`, `BUILDKITE_COMMIT`, `BUILDKITE_BUILD_NUMBER`, `BUILDKITE_BUILD_URL` |
| Outputs | GitHub commit status `beats-tests (daily)` on every covered commit; no artifacts. |
| Permissions | The publish step needs `statuses: write` on `elastic/beats` for the pipeline's ephemeral GitHub token. That is granted in Elastic's internal CI configuration, not in this repository. |
| Secrets | `GITHUB_TOKEN`, exported by the `elastic/vault-github-token` plugin from the `VAULT_GITHUB_TOKEN` that the agent `environment` hook injects. |

## Operational notes

Every trigger step needs a key starting with `test-`. The publish step reads each one's outcome with `buildkite-agent step get outcome`, and fails if the number of keys does not match the number of trigger steps.

The status state is `failure` if any triggered build failed, `error` if a trigger step reports any other outcome than passed or failed, and `success` otherwise. A canceled build publishes nothing, because `continue_on_failure` does not continue past a cancelation. The status description names the build number, the tested commit and the failing Beats, and its link points to this build.

The covered commits are the tested commit plus its first-parent ancestors back to, but not including, the first commit that a build of a different commit already marked. When the tested commit was already tested, for example by daily builds over a weekend, the ancestors that earlier build marked are marked again with the new result. The script looks at the last 100 commits; if it finds no boundary, for example on the first run, it only marks the tested commit. A commit with a `failure` status was not necessarily the cause: the failure came from somewhere in the covered range, or was already failing before it.

Retrying failed jobs in a triggered build updates the trigger step once the triggered build finishes. Retry the publish step afterwards to publish the new result; it marks the same commits again. It publishes nothing if a newer `beats-tests` build already marked the tested commit. Commits are published oldest first, so if a request fails, the tested commit stays unmarked and the next build covers the commits left behind.

`DRY_RUN=true` prints the covered commits and the status without publishing. It still queries GitHub, so it needs `gh` read access, a `buildkite-agent` that answers `step get outcome`, and `BUILDKITE_COMMIT`, `BUILDKITE_BUILD_NUMBER` and `BUILDKITE_BUILD_URL`.

## Sources

- Pipeline: [`.buildkite/beats-tests-pipeline.yml`](../beats-tests-pipeline.yml)
- Script: [`.buildkite/scripts/daily-tests-commit-status.sh`](../scripts/daily-tests-commit-status.sh)
