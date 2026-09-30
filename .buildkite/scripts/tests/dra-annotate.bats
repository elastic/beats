#!/usr/bin/env bats
##
##  Tests for .buildkite/scripts/dra-annotate.sh.
##
##  The manifest fixtures follow the dractl manifest schema (2.1.0), where
##  `prefix` is the product id and the file is written to
##  artifacts/dra/<product>/<build_id>/manifest-<version>.json.
##

load helpers

setup() {
  dra_test_setup
  SCRIPT="$REPO_ROOT/.buildkite/scripts/dra-annotate.sh"
  PIPELINE="$REPO_ROOT/.buildkite/dra-prep-pipeline.yml"
  cd "$TEST_TMPDIR/work" || return 1
}

teardown() {
  dra_test_teardown
}

# Prints the prep step key the annotate step depends on, as declared in
# dra-prep-pipeline.yml, with ${WORKFLOW} substituted.
prep_step_key() {
  local workflow="$1" key
  key=$(yq -r '.steps[] | select(.command == "*dra-annotate.sh*") | .depends_on' "$PIPELINE")
  printf '%s\n' "${key//\$\{WORKFLOW\}/$workflow}"
}

# write_manifest <build_id> <version> [jq filter applied to the manifest]
write_manifest() {
  local build_id="$1" version="$2" filter="${3:-.}" dir
  dir="$MOCK_ARTIFACT_ROOT/artifacts/dra/beats/$build_id"
  mkdir -p "$dir"
  jq -n --arg build_id "$build_id" --arg version "$version" '{
    manifest_version: "2.1.0",
    build_id: $build_id,
    prefix: "beats",
    version: $version,
    branch: "9.9",
    release_branch: "9.9",
    start_time: "Thu, 25 Sep 2026 10:00:00 GMT",
    end_time: "Thu, 25 Sep 2026 10:05:00 GMT",
    build_duration_seconds: 300,
    projects: {beats: {branch: "9.9", build_duration_seconds: 0, packages: {}}}
  }' | jq "$filter" >"$dir/manifest-$version.json"
}

assert_not_annotated() {
  [ ! -e "$DRA_ANNOTATION_OUTPUT" ]
  ! grep -q '^annotate' "$BUILDKITE_AGENT_LOG" 2>/dev/null
}

@test "pipeline annotate step depends on a declared prep step key" {
  local key
  key=$(yq -r '.steps[] | select(.command == "*dra-annotate.sh*") | .depends_on' "$PIPELINE")
  [ -n "$key" ] && [ "$key" != "null" ]
  yq -e ".steps[] | select(.key == \"$key\")" "$PIPELINE" >/dev/null
}

@test "snapshot annotates the summary link" {
  MOCK_ARTIFACT_STEP=$(prep_step_key snapshot)
  export MOCK_ARTIFACT_STEP
  write_manifest "9.9.0-ab12cd34" "9.9.0-SNAPSHOT"

  run "$SCRIPT" snapshot
  [ "$status" -eq 0 ]

  url="https://artifacts-snapshot.elastic.co/beats/9.9.0-ab12cd34/summary-9.9.0-SNAPSHOT.html"
  [ "$(cat "$DRA_ANNOTATION_OUTPUT")" = "**snapshot summary link:** [$url]($url)" ]
}

@test "staging annotates the summary link" {
  MOCK_ARTIFACT_STEP=$(prep_step_key staging)
  export MOCK_ARTIFACT_STEP
  write_manifest "9.9.0-ef56ab78" "9.9.0"

  run "$SCRIPT" staging
  [ "$status" -eq 0 ]

  url="https://artifacts-staging.elastic.co/beats/9.9.0-ef56ab78/summary-9.9.0.html"
  [ "$(cat "$DRA_ANNOTATION_OUTPUT")" = "**staging summary link:** [$url]($url)" ]
}

@test "staging with a version qualifier annotates the qualified summary link" {
  MOCK_ARTIFACT_STEP=$(prep_step_key staging)
  export MOCK_ARTIFACT_STEP
  write_manifest "9.9.0-ef56ab78" "9.9.0-beta1"

  run "$SCRIPT" staging
  [ "$status" -eq 0 ]

  url="https://artifacts-staging.elastic.co/beats/9.9.0-ef56ab78/summary-9.9.0-beta1.html"
  [ "$(cat "$DRA_ANNOTATION_OUTPUT")" = "**staging summary link:** [$url]($url)" ]
}

@test "prefix leading and trailing slashes are stripped" {
  MOCK_ARTIFACT_STEP=$(prep_step_key snapshot)
  export MOCK_ARTIFACT_STEP
  write_manifest "9.9.0-ab12cd34" "9.9.0-SNAPSHOT" '.prefix = "/beats/"'

  run "$SCRIPT" snapshot
  [ "$status" -eq 0 ]

  url="https://artifacts-snapshot.elastic.co/beats/9.9.0-ab12cd34/summary-9.9.0-SNAPSHOT.html"
  grep -qF "[$url]($url)" "$DRA_ANNOTATION_OUTPUT"
  run sed 's|https://||g' "$DRA_ANNOTATION_OUTPUT"
  [[ "$output" != *"//"* ]]
}

@test "downloads the manifest glob from the prep step" {
  write_manifest "9.9.0-ab12cd34" "9.9.0-SNAPSHOT"

  run "$SCRIPT" snapshot
  [ "$status" -eq 0 ]

  grep -qxF "artifact download artifacts/dra/beats/*/manifest-*.json . --step $(prep_step_key snapshot)" \
    "$BUILDKITE_AGENT_LOG"
}

@test "annotates with success style and appends" {
  write_manifest "9.9.0-ab12cd34" "9.9.0-SNAPSHOT"

  run "$SCRIPT" snapshot
  [ "$status" -eq 0 ]

  grep -qxF "annotate --style=success --append" "$BUILDKITE_AGENT_LOG"
}

@test "fails when the prep step key does not match" {
  export MOCK_ARTIFACT_STEP="dra-prep-staging"
  write_manifest "9.9.0-ab12cd34" "9.9.0-SNAPSHOT"

  run "$SCRIPT" snapshot
  [ "$status" -ne 0 ]
  [[ "$output" == *"no step found"* ]]
  assert_not_annotated
}

@test "fails when the prep step uploaded no manifest" {
  MOCK_ARTIFACT_STEP=$(prep_step_key snapshot)
  export MOCK_ARTIFACT_STEP

  run "$SCRIPT" snapshot
  [ "$status" -ne 0 ]
  [[ "$output" == *"no artifacts found"* ]]
  assert_not_annotated
}

@test "fails when manifest is missing prefix" {
  write_manifest "9.9.0-ab12cd34" "9.9.0-SNAPSHOT" 'del(.prefix)'

  run "$SCRIPT" snapshot
  [ "$status" -ne 0 ]
  assert_not_annotated
}

@test "fails when manifest is missing build_id" {
  write_manifest "9.9.0-ab12cd34" "9.9.0-SNAPSHOT" 'del(.build_id)'

  run "$SCRIPT" snapshot
  [ "$status" -ne 0 ]
  assert_not_annotated
}

@test "fails when manifest is missing version" {
  write_manifest "9.9.0-ab12cd34" "9.9.0-SNAPSHOT" 'del(.version)'

  run "$SCRIPT" snapshot
  [ "$status" -ne 0 ]
  assert_not_annotated
}

@test "fails when manifest fields are null" {
  for field in prefix build_id version; do
    rm -rf "$MOCK_ARTIFACT_ROOT/artifacts" "$TEST_TMPDIR/work/artifacts" "$DRA_ANNOTATION_OUTPUT"
    write_manifest "9.9.0-ab12cd34" "9.9.0-SNAPSHOT" ".${field} = null"

    run "$SCRIPT" snapshot
    [ "$status" -ne 0 ]
    [ ! -e "$DRA_ANNOTATION_OUTPUT" ]
  done
  assert_not_annotated
}

@test "fails when the workflow argument is missing" {
  write_manifest "9.9.0-ab12cd34" "9.9.0-SNAPSHOT"

  run "$SCRIPT"
  [ "$status" -ne 0 ]
  [[ "$output" == *"workflow required"* ]]
  [ ! -e "$BUILDKITE_AGENT_LOG" ]
}

# With several manifests in the download dir the script takes whichever
# `find` lists first, without checking the workflow. Not tested: `--step
# dra-prep-<workflow>` scopes the download to the single prep job of that
# workflow, and each job starts from a clean checkout, so a manifest from the
# other workflow cannot be present.
