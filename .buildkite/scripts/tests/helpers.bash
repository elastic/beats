#!/usr/bin/env bash
##
##  Shared setup for the DRA Bats suites. Stubs make, curl, and
##  buildkite-agent on PATH so the DRA scripts run without packaging,
##  network access, plugins, or publishing.
##
##  Stub behavior is driven by env vars:
##    DRA_TEST_VERSION       value printed by `make get-version`
##    MOCK_GCS_QUALIFIER     qualifier served by the dra-qualifier bucket;
##                           when unset, the bucket lookup returns 404
##    MOCK_ARTIFACT_ROOT     tree served by `buildkite-agent artifact
##                           download`; only paths matching the requested
##                           glob are copied, and no match is an error
##    MOCK_ARTIFACT_STEP     when set, artifact download fails unless
##                           `--step` matches it
##

dra_test_setup() {
  REPO_ROOT=$(cd "$BATS_TEST_DIRNAME/../../.." && pwd)
  export REPO_ROOT
  TEST_TMPDIR=$(mktemp -d)
  mkdir -p "$TEST_TMPDIR/bin" "$TEST_TMPDIR/work" "$TEST_TMPDIR/artifact-root"

  cat >"$TEST_TMPDIR/bin/make" <<'MOCK'
#!/usr/bin/env bash
if [[ "$*" != "get-version" ]]; then
  echo "unexpected make invocation: $*" >&2
  exit 1
fi
printf '%s\n' "${DRA_TEST_VERSION:?}"
MOCK

  cat >"$TEST_TMPDIR/bin/curl" <<'MOCK'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${CURL_LOG:?}"
if [[ -z "${MOCK_GCS_QUALIFIER+x}" ]]; then
  exit 22
fi
for arg in "$@"; do
  [[ "$arg" == "-o" ]] && exit 0
done
printf '%s' "${MOCK_GCS_QUALIFIER}"
MOCK

  cat >"$TEST_TMPDIR/bin/buildkite-agent" <<'MOCK'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${BUILDKITE_AGENT_LOG:?}"
case "${1:-} ${2:-}" in
  "pipeline upload")
    # Mimic Buildkite's upload-time interpolation: the YAML is parsed first
    # and only string scalars are interpolated against the uploading
    # process's environment, so `upload: ${VAR}` stays a string.
    yq '(.. | select(tag == "!!str")) |= envsubst' "${3:-.buildkite/pipeline.yml}" >"${PIPELINE_OUTPUT:?}"
    ;;
  "artifact download")
    glob="$3"
    dest="${4:-.}"
    step=""
    shift 4 || true
    while [[ $# -gt 0 ]]; do
      [[ "$1" == "--step" ]] && step="$2"
      shift
    done
    if [[ -n "${MOCK_ARTIFACT_STEP:-}" && "$step" != "${MOCK_ARTIFACT_STEP}" ]]; then
      echo "fatal: no step found matching \"${step}\"" >&2
      exit 1
    fi
    found=false
    while IFS= read -r -d '' file; do
      rel="${file#"${MOCK_ARTIFACT_ROOT:?}"/}"
      # shellcheck disable=SC2053 # glob match is intended
      if [[ "$rel" == $glob ]]; then
        mkdir -p "$dest/$(dirname "$rel")"
        cp "$file" "$dest/$rel"
        found=true
      fi
    done < <(find "${MOCK_ARTIFACT_ROOT:?}" -type f -print0)
    if [[ "$found" != "true" ]]; then
      echo "fatal: no artifacts found for search \"${glob}\"" >&2
      exit 1
    fi
    ;;
  "annotate "*)
    cat >"${DRA_ANNOTATION_OUTPUT:?}"
    ;;
  *)
    echo "unexpected buildkite-agent invocation: $*" >&2
    exit 1
    ;;
esac
MOCK
  chmod +x "$TEST_TMPDIR"/bin/*

  export PATH="$TEST_TMPDIR/bin:$PATH"
  export BUILDKITE_AGENT_LOG="$TEST_TMPDIR/buildkite-agent.log"
  export CURL_LOG="$TEST_TMPDIR/curl.log"
  export PIPELINE_OUTPUT="$TEST_TMPDIR/pipeline.yml"
  export DRA_ANNOTATION_OUTPUT="$TEST_TMPDIR/annotation.md"
  export MOCK_ARTIFACT_ROOT="$TEST_TMPDIR/artifact-root"
  export DRA_TEST_VERSION="9.9.0"
  export BUILDKITE_BRANCH="9.9"
  export BUILDKITE_PIPELINE_SLUG="beats-packaging-pipeline"
  export BUILDKITE_BUILD_NUMBER="1234"
  unset DRY_RUN DRA_BRANCH VERSION_QUALIFIER MOCK_GCS_QUALIFIER MOCK_ARTIFACT_STEP \
    WORKFLOW DRA_WORKFLOW STACK_VERSION DRA_UPLOAD
}

dra_test_teardown() {
  rm -rf "$TEST_TMPDIR"
}
