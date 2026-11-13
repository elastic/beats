# Beats Release Automation

Release-manager guide for Beats feature-freeze and patch releases using mage.

For commands, environment variables, file lists, and troubleshooting, see
[dev-tools/mage/release/README.md](dev-tools/mage/release/README.md).

## When to run which workflow

| Situation | Version shape | Command |
|-----------|---------------|---------|
| Feature freeze / major or minor release | `X.Y.0` (e.g. `9.5.0`) | `mage release:runMajorMinor` |
| Patch release on an existing branch | `X.Y.Z` with `Z > 0` (e.g. `9.2.1`) | `mage release:runPatch` |
| Refresh the Beats release checklist issue only | any `CURRENT_RELEASE` | `mage release:ensureIssueTracker` |

Minor feature-freeze releases for 6.x, 7.x, and 8.x are blocked. Patch releases
are allowed for all supported versions.

Release notes are **not** produced by mage. Use
[`.github/workflows/release-notes.yml`](.github/workflows/release-notes.yml)
separately.

## Quick Start

Always prefer `DRY_RUN=true` first. `GITHUB_TOKEN` is required only when not in
dry-run mode.

### Feature freeze (e.g. 9.5.0)

```bash
export CURRENT_RELEASE="9.5.0"
export GITHUB_TOKEN="ghp_your_token"
export DRY_RUN=true

mage release:runMajorMinor

# Review local branches and diffs, then:
export DRY_RUN=false
mage release:runMajorMinor
```

### Patch release (e.g. 9.2.1)

```bash
git checkout 9.2
git pull

export CURRENT_RELEASE="9.2.1"
export BASE_BRANCH="9.2"
export GITHUB_TOKEN="ghp_your_token"
export DRY_RUN=true

mage release:runPatch

# Review, then:
export DRY_RUN=false
mage release:runPatch
```

## What feature freeze produces

Creates the release branch (e.g. `9.5` from `main`) and opens **3 grouped PRs**.

`CURRENT_RELEASE` must already match `libbeat/version/version.go` on `BASE_BRANCH`
(set by the previous cycle's next-minor version bump). The workflow fails fast if
they differ. Next-patch bumps are **not** opened here; use `runPatch` after the
patch release trigger.

Merge order matters; labels encode timing. Titles are prefixed with
`[Release <CURRENT_RELEASE>]`.

| Order | PR | Target | Merge label | Purpose |
|-------|-----|--------|-------------|---------|
| 1 | PR-A | `main` | `merge:1-ff-day` | Mergify backport rule + bump `version.go` to next minor |
| 2 | PR-B | release branch | `merge:2-after-branch` | Feature-freeze version, docs, test env, `make update` |
| 3 | PR-C | `main` | `merge:3-after-images` | Docs + test env for the next minor |

**RM merge order:** push release branch → merge PR-A → merge PR-B → merge PR-C.

## What a patch release produces

Runs on the existing release branch and opens **1 PR**: the post-release version
bump to `CURRENT_RELEASE` (`NEW_VERSION` from version-bump), plus Beats test-env.
Docs stay off this path.

`CURRENT_RELEASE` must have patch `> 0`, and `version.go` on the release branch
must equal the previous patch on the same line. The workflow fails fast otherwise.

| Order | PR | Target | Merge label | Purpose |
|-------|-----|--------|-------------|---------|
| 1 | bump | release branch | `merge:4-after-release` | `version.go` → CURRENT + test-env (`LATEST=prev`) |

Titles are prefixed with `[Release <CURRENT_RELEASE>]`.

**RM merge order:** merge the bump PR promptly after the patch job opens it.
Release notes stay on the separate GitHub Actions workflow.

Both `runMajorMinor` and `runPatch` finish by ensuring a Beats issue titled
`[RELEASE <CURRENT_RELEASE>] Instructions & Checklist` exists (create or update).
The issue links https://github.com/elastic/ingest-dev/issues/8866 and lists
related Beats PRs with label `release` for that version. That step is
**best-effort**: a tracker failure is logged as a warning and does not fail the
release workflow. Re-run anytime with `mage release:ensureIssueTracker`.

## Release manager checklist

1. Set `CURRENT_RELEASE` (and `BASE_BRANCH` for patches).
2. Run with `DRY_RUN=true` and review branches / `git diff`.
3. Re-run with `DRY_RUN=false` to push and open PRs (also creates/updates the release issue tracker).
4. Merge PRs in label order (`merge:1-…` before `merge:2-…`, and so on).
5. Generate release notes via the separate GitHub Actions workflow when needed.
6. Optionally re-run `mage release:ensureIssueTracker` if more `release`-labeled PRs appear later.

## More detail

Operator and developer documentation (full env table, individual mage targets,
DRY_RUN behavior, files touched, FAQ, troubleshooting):

→ [dev-tools/mage/release/README.md](dev-tools/mage/release/README.md)
