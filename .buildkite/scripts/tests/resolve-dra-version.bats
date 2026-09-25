#!/usr/bin/env bats
##
##  Tests for .buildkite/scripts/packaging/resolve-dra-version.sh, which the
##  packaging step sources to resolve STACK_VERSION, VERSION_QUALIFIER, and
##  DRA_UPLOAD before uploading dra-prep-pipeline.yml.
##
##  Each test sources the script in a subshell and prints the resolved values
##  from a child process, so only exported variables are observed.
##

# shellcheck disable=SC2030,SC2031 # bats runs each @test in a subshell by design
load helpers

SCRIPT=.buildkite/scripts/packaging/resolve-dra-version.sh

setup() {
  dra_test_setup
  cd "$REPO_ROOT" || return
}

teardown() {
  dra_test_teardown
}

# Sources the script and prints STACK_VERSION|VERSION_QUALIFIER|DRA_UPLOAD as
# seen by a child process, i.e. only what was exported.
resolve() {
  run bash -c '
    source "$1"
    bash -c "printf \"%s|%s|%s\n\" \"\${STACK_VERSION-<unset>}\" \"\${VERSION_QUALIFIER-<unset>}\" \"\${DRA_UPLOAD-<unset>}\""
  ' _ "$SCRIPT"
}

curl_calls() {
  if [[ -f "$CURL_LOG" ]]; then
    wc -l <"$CURL_LOG"
  else
    echo 0
  fi
}

@test "snapshot resolves the plain version and exports all variables" {
  export WORKFLOW=snapshot
  resolve
  [ "$status" -eq 0 ]
  [ "${lines[-1]}" = "9.9.0||true" ]
  [ "$(curl_calls)" -eq 0 ]
}

@test "snapshot ignores an externally set VERSION_QUALIFIER for STACK_VERSION" {
  export WORKFLOW=snapshot
  export VERSION_QUALIFIER=beta1
  resolve
  [ "$status" -eq 0 ]
  [ "${lines[-1]}" = "9.9.0|beta1|true" ]
  [ "$(curl_calls)" -eq 0 ]
}

@test "snapshot never queries the qualifier bucket even when it has a qualifier" {
  export WORKFLOW=snapshot
  export MOCK_GCS_QUALIFIER=beta1
  resolve
  [ "$status" -eq 0 ]
  [ "${lines[-1]}" = "9.9.0||true" ]
  [ "$(curl_calls)" -eq 0 ]
}

@test "staging with an externally set VERSION_QUALIFIER qualifies without querying the bucket" {
  export WORKFLOW=staging
  export VERSION_QUALIFIER=rc2
  export MOCK_GCS_QUALIFIER=beta1
  resolve
  [ "$status" -eq 0 ]
  [ "${lines[-1]}" = "9.9.0-rc2|rc2|true" ]
  [ "$(curl_calls)" -eq 0 ]
}

@test "staging falls back to the plain version when the bucket returns 404" {
  export WORKFLOW=staging
  resolve
  [ "$status" -eq 0 ]
  [ "${lines[-1]}" = "9.9.0||true" ]
  [ "$(curl_calls)" -eq 1 ]
}

@test "staging qualifies the version with the qualifier served by the bucket" {
  export WORKFLOW=staging
  export MOCK_GCS_QUALIFIER=beta1
  resolve
  [ "$status" -eq 0 ]
  [ "${lines[-1]}" = "9.9.0-beta1|beta1|true" ]
  [ "$(curl_calls)" -gt 0 ]
  while IFS= read -r call; do
    [[ "$call" == *"https://storage.googleapis.com/dra-qualifier/${BUILDKITE_BRANCH}" ]]
  done <"$CURL_LOG"
}

@test "staging looks up the qualifier for DRA_BRANCH when it is set" {
  export WORKFLOW=staging
  export DRA_BRANCH=main
  export MOCK_GCS_QUALIFIER=alpha1
  resolve
  [ "$status" -eq 0 ]
  [ "${lines[-1]}" = "9.9.0-alpha1|alpha1|true" ]
  [ "$(curl_calls)" -gt 0 ]
  while IFS= read -r call; do
    [[ "$call" == *"/dra-qualifier/main" ]]
  done <"$CURL_LOG"
}

@test "staging uses the version reported by make get-version" {
  export WORKFLOW=staging
  export DRA_TEST_VERSION=10.1.0
  export MOCK_GCS_QUALIFIER=rc1
  resolve
  [ "$status" -eq 0 ]
  [ "${lines[-1]}" = "10.1.0-rc1|rc1|true" ]
}

@test "DRY_RUN=true disables the DRA upload" {
  export WORKFLOW=snapshot
  export DRY_RUN=true
  resolve
  [ "$status" -eq 0 ]
  [ "${lines[-1]}" = "9.9.0||false" ]
}

@test "DRY_RUN unset, false, or any other value keeps the DRA upload enabled" {
  export WORKFLOW=snapshot
  for value in "<unset>" false "" TRUE yes 1; do
    if [[ "$value" == "<unset>" ]]; then
      unset DRY_RUN
    else
      export DRY_RUN="$value"
    fi
    resolve
    [ "$status" -eq 0 ]
    [ "${lines[-1]}" = "9.9.0||true" ]
  done
}

@test "staging works under set -u on a fresh job where VERSION_QUALIFIER was never set" {
  export WORKFLOW=staging
  export MOCK_GCS_QUALIFIER=beta1
  run bash -c '
    set -u
    [[ -z "${VERSION_QUALIFIER+x}" ]] || exit 99
    source "$1"
    bash -c "printf \"%s|%s|%s\n\" \"\$STACK_VERSION\" \"\$VERSION_QUALIFIER\" \"\$DRA_UPLOAD\""
  ' _ "$SCRIPT"
  [ "$status" -eq 0 ]
  [ "${lines[-1]}" = "9.9.0-beta1|beta1|true" ]
}

@test "staging works under set -u when WORKFLOW, DRY_RUN, and BUILDKITE_BRANCH are unset" {
  unset BUILDKITE_BRANCH
  export WORKFLOW=staging
  run bash -c '
    set -u
    source "$1"
    bash -c "printf \"%s|%s|%s\n\" \"\$STACK_VERSION\" \"\$VERSION_QUALIFIER\" \"\$DRA_UPLOAD\""
  ' _ "$SCRIPT"
  [ "$status" -eq 0 ]
  [ "${lines[-1]}" = "9.9.0||true" ]
}

@test "an unknown or missing WORKFLOW is treated like snapshot" {
  export MOCK_GCS_QUALIFIER=beta1
  resolve
  [ "$status" -eq 0 ]
  [ "${lines[-1]}" = "9.9.0||true" ]
  [ "$(curl_calls)" -eq 0 ]
}

@test "a make get-version failure aborts sourcing with a non-zero exit" {
  cat >"$TEST_TMPDIR/bin/make" <<'MOCK'
#!/usr/bin/env bash
echo "make: *** No rule to make target 'get-version'." >&2
exit 2
MOCK
  chmod +x "$TEST_TMPDIR/bin/make"
  export WORKFLOW=staging
  export MOCK_GCS_QUALIFIER=beta1
  run bash -c 'source "$1"; echo "should not be reached"' _ "$SCRIPT"
  [ "$status" -ne 0 ]
  [[ "$output" != *"should not be reached"* ]]
  [ "$(curl_calls)" -eq 0 ]
}
