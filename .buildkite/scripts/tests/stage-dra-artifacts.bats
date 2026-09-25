#!/usr/bin/env bats
##
##  Tests for .buildkite/scripts/stage-dra-artifacts.sh: artifact download,
##  dependencies CSV and docker tarball renames, and staging of the
##  workflow's slice into artifacts/ for elastic/dra-prep-buildkite-plugin.
##

load helpers

setup() {
  dra_test_setup
  DIST="$MOCK_ARTIFACT_ROOT/build/distributions"
}

teardown() {
  dra_test_teardown
}

run_stage() {
  run bash -c 'cd "$1" && bash "$2/.buildkite/scripts/stage-dra-artifacts.sh"' \
    _ "$TEST_TMPDIR/work" "$REPO_ROOT"
}

# Prints "<subdir>/<basename>" for every file package-dra.sh and the
# dashboards step would upload for one package version (e.g. 9.9.0-SNAPSHOT
# or 9.9.0-beta1). Naming follows dev-tools/mage/pkgtypes.go
# (defaultBinaryName) and dev-tools/packaging/packages.yml. filebeat/ holds
# both x-pack (filebeat-*) and OSS (filebeat-oss-*) packages because
# package-dra.sh strips "x-pack/" from the destination subdir.
fixture_files() {
  local v="$1" beat name image f
  for beat in filebeat metricbeat; do
    for name in "$beat" "$beat-oss"; do
      for f in \
        "$name-$v-linux-x86_64.tar.gz" \
        "$name-$v-linux-arm64.tar.gz" \
        "$name-$v-darwin-aarch64.tar.gz" \
        "$name-$v-windows-x86_64.zip" \
        "$name-$v-amd64.deb" \
        "$name-$v-arm64.deb" \
        "$name-$v-x86_64.rpm" \
        "$name-$v-aarch64.rpm"; do
        printf '%s\n' "$beat/$f" "$beat/$f.sha512"
      done
    done
    printf '%s\n' "$beat/$beat-fips-$v-linux-x86_64.tar.gz" \
      "$beat/$beat-fips-$v-linux-x86_64.tar.gz.sha512"
    for image in "$beat" "$beat-wolfi" "$beat-oss"; do
      for f in \
        "$image-$v-linux-amd64.docker.tar.gz" \
        "$image-$v-linux-arm64.docker.tar.gz"; do
        printf '%s\n' "$beat/$f" "$beat/$f.sha512"
      done
    done
  done
  printf '%s\n' "dashboards/beats-dashboards-$v.zip" \
    "dashboards/beats-dashboards-$v.zip.sha512"
}

# Writes the artifact tree for the given package versions, plus the single
# top-level dependencies.csv, under $MOCK_ARTIFACT_ROOT.
make_fixture() {
  local v rel
  mkdir -p "$DIST"
  for v in "$@"; do
    while IFS= read -r rel; do
      mkdir -p "$DIST/$(dirname "$rel")"
      printf '%s\n' "$rel" >"$DIST/$rel"
    done < <(fixture_files "$v")
  done
  printf 'name,version\ngithub.com/foo/bar,v1.0.0\n' >"$DIST/dependencies.csv"
}

# Expected artifacts/ listing for one package version: basenames with the
# docker renames applied, plus the renamed dependencies CSV.
expected_artifacts() {
  local v="$1" csv_version="$2"
  {
    fixture_files "$v" | xargs -n1 basename |
      sed -e 's/linux-amd64\.docker\.tar\.gz/docker-image-linux-amd64.tar.gz/' \
          -e 's/linux-arm64\.docker\.tar\.gz/docker-image-linux-arm64.tar.gz/'
    printf 'dependencies-%s.csv\n' "$csv_version"
  } | LC_ALL=C sort
}

staged_artifacts() {
  (cd "$TEST_TMPDIR/work/artifacts" && ls -1A) | LC_ALL=C sort
}

@test "downloads build/**/* from this build" {
  make_fixture 9.9.0-SNAPSHOT 9.9.0
  export DRA_WORKFLOW=snapshot

  run_stage

  [ "$status" -eq 0 ]
  grep -qxF 'artifact download build/**/* .' "$BUILDKITE_AGENT_LOG"
}

@test "snapshot: stages exactly the SNAPSHOT files" {
  make_fixture 9.9.0-SNAPSHOT 9.9.0
  export DRA_WORKFLOW=snapshot

  run_stage

  [ "$status" -eq 0 ]
  diff <(expected_artifacts 9.9.0-SNAPSHOT 9.9.0-SNAPSHOT) <(staged_artifacts)
  [ -f "$TEST_TMPDIR/work/artifacts/dependencies-9.9.0-SNAPSHOT.csv" ]
  [ ! -e "$TEST_TMPDIR/work/artifacts/dependencies.csv" ]
  [ -z "$(staged_artifacts | grep -v -- '-SNAPSHOT')" ]
}

@test "staging: stages exactly the non-SNAPSHOT files" {
  make_fixture 9.9.0-SNAPSHOT 9.9.0
  export DRA_WORKFLOW=staging

  run_stage

  [ "$status" -eq 0 ]
  diff <(expected_artifacts 9.9.0 9.9.0) <(staged_artifacts)
  [ -f "$TEST_TMPDIR/work/artifacts/dependencies-9.9.0.csv" ]
  [ ! -e "$TEST_TMPDIR/work/artifacts/dependencies.csv" ]
  # no SNAPSHOT leak, including .sha512 siblings
  [ -z "$(staged_artifacts | grep -- 'SNAPSHOT')" ]
  [ ! -e "$TEST_TMPDIR/work/artifacts/filebeat-9.9.0-SNAPSHOT-linux-x86_64.tar.gz.sha512" ]
  [ ! -e "$TEST_TMPDIR/work/artifacts/filebeat-9.9.0-SNAPSHOT-docker-image-linux-amd64.tar.gz.sha512" ]
}

@test "staging with VERSION_QUALIFIER: stages qualifier-named packages" {
  # A qualifier build only runs staging (snapshot is gated off).
  make_fixture 9.9.0-beta1
  export DRA_WORKFLOW=staging VERSION_QUALIFIER=beta1

  run_stage

  [ "$status" -eq 0 ]
  diff <(expected_artifacts 9.9.0-beta1 9.9.0-beta1) <(staged_artifacts)
  [ -f "$TEST_TMPDIR/work/artifacts/dependencies-9.9.0-beta1.csv" ]
  [ -f "$TEST_TMPDIR/work/artifacts/filebeat-9.9.0-beta1-linux-x86_64.tar.gz" ]
  [ -f "$TEST_TMPDIR/work/artifacts/filebeat-9.9.0-beta1-docker-image-linux-arm64.tar.gz.sha512" ]
}

@test "snapshot with VERSION_QUALIFIER: CSV name matches mage package naming" {
  skip "BUG: CSV is named dependencies-9.9.0-SNAPSHOT-beta1.csv, but mage names snapshot packages 9.9.0-beta1-SNAPSHOT (latent: packaging.pipeline.yml gates snapshot on VERSION_QUALIFIER == null)"
  make_fixture 9.9.0-beta1-SNAPSHOT
  export DRA_WORKFLOW=snapshot VERSION_QUALIFIER=beta1

  run_stage

  [ "$status" -eq 0 ]
  diff <(expected_artifacts 9.9.0-beta1-SNAPSHOT 9.9.0-beta1-SNAPSHOT) <(staged_artifacts)
}

@test "renames docker tarballs and their .sha512 siblings" {
  make_fixture 9.9.0-SNAPSHOT 9.9.0
  export DRA_WORKFLOW=staging

  run_stage

  [ "$status" -eq 0 ]
  local a="$TEST_TMPDIR/work/artifacts"
  for image in filebeat filebeat-wolfi filebeat-oss metricbeat; do
    for arch in amd64 arm64; do
      [ -f "$a/$image-9.9.0-docker-image-linux-$arch.tar.gz" ]
      [ -f "$a/$image-9.9.0-docker-image-linux-$arch.tar.gz.sha512" ]
    done
  done
  [ -z "$(staged_artifacts | grep -F '.docker.tar.gz')" ]
  # also renamed in place in build/distributions (snapshot files included)
  [ -f "$TEST_TMPDIR/work/build/distributions/filebeat/filebeat-9.9.0-SNAPSHOT-docker-image-linux-arm64.tar.gz.sha512" ]
  [ -z "$(find "$TEST_TMPDIR/work/build/distributions" -name '*.docker.tar.gz*')" ]
}

@test "flattens artifacts into artifacts/ without losing files" {
  make_fixture 9.9.0-SNAPSHOT 9.9.0
  export DRA_WORKFLOW=snapshot

  run_stage

  [ "$status" -eq 0 ]
  [ -z "$(find "$TEST_TMPDIR/work/artifacts" -mindepth 1 -type d)" ]
  # cp into a flat dir silently overwrites on basename collisions; the
  # realistic fixture spans several subdirs, so every source must survive.
  local sources staged
  sources=$(find "$TEST_TMPDIR/work/build/distributions" -type f -name '*-SNAPSHOT*' | wc -l)
  staged=$(staged_artifacts | wc -l)
  [ "$sources" -eq "$staged" ]
  [ -z "$(find "$TEST_TMPDIR/work/build/distributions" -type f -printf '%f\n' | sort | uniq -d)" ]
}

@test "fails when no files match the workflow" {
  skip "BUG: the renamed dependencies CSV always matches the workflow, so the 'no artifacts found' guard never fires and a staging run with only SNAPSHOT packages succeeds staging just the CSV"
  make_fixture 9.9.0-SNAPSHOT
  export DRA_WORKFLOW=staging

  run_stage

  [ "$status" -ne 0 ]
  [[ "$output" == *"ERROR: no staging artifacts found"* ]]
}

@test "fails when dependencies.csv is missing" {
  make_fixture 9.9.0-SNAPSHOT 9.9.0
  rm "$DIST/dependencies.csv"
  export DRA_WORKFLOW=snapshot

  run_stage

  [ "$status" -ne 0 ]
  [[ "$output" == *"build/distributions/dependencies.csv"* ]]
  [ ! -e "$TEST_TMPDIR/work/artifacts" ]
}

@test "fails when DRA_WORKFLOW is unset" {
  make_fixture 9.9.0-SNAPSHOT 9.9.0

  run_stage

  [ "$status" -ne 0 ]
  [[ "$output" == *"DRA_WORKFLOW is required"* ]]
  [ ! -s "$BUILDKITE_AGENT_LOG" ]
}
