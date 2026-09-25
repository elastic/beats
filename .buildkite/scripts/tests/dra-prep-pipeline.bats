#!/usr/bin/env bats
##
##  End-to-end checks for the DRA sub-pipeline. Runs the real dra-snapshot /
##  dra-staging step commands from packaging.pipeline.yml against the stubbed
##  buildkite-agent and asserts on the rendered dra-prep-pipeline.yml.
##

load helpers

PACKAGING_PIPELINE=".buildkite/packaging.pipeline.yml"
DRA_PIPELINE=".buildkite/dra-prep-pipeline.yml"

setup() {
  dra_test_setup
  cd "$REPO_ROOT"
}

teardown() {
  dra_test_teardown
}

# Prints a field of a step in the DRA publish group, e.g. dra_step dra-snapshot .command
dra_step() {
  local key="$1" path="$2"
  yq "explode(.) | .steps[] | select(.group == \"DRA publish\") | .steps[] | select(.key == \"${key}\") | ${path}" "$PACKAGING_PIPELINE"
}

# Runs the real command of the dra-<workflow> step with its WORKFLOW env and
# writes the rendered pipeline to $PIPELINE_OUTPUT. Extra env can be passed
# by the caller via exported vars.
render_workflow() {
  local workflow="$1" cmd step_workflow
  cmd=$(dra_step "dra-${workflow}" .command)
  step_workflow=$(dra_step "dra-${workflow}" .env.WORKFLOW)
  [[ -n "$cmd" && "$cmd" != "null" ]] || { echo "no command for dra-${workflow}" >&2; return 1; }
  [[ "$step_workflow" == "$workflow" ]] || { echo "unexpected WORKFLOW env: $step_workflow" >&2; return 1; }
  WORKFLOW="$step_workflow" bash -c "$cmd"
}

# Prints the full plugin reference (name#version) matching a plugin name.
plugin_ref() {
  local file="$1" name="$2"
  yq ".steps[0].plugins[] | keys | .[] | select(test(\"^${name}#\"))" "$file"
}

# Query helpers over the rendered pipeline.
rendered() { yq "$1" "$PIPELINE_OUTPUT"; }
prep() { rendered ".steps[] | select(.command == \".buildkite/scripts/stage-dra-artifacts.sh\") | $1"; }
annotate() { rendered ".steps[] | select(.command | test(\"dra-annotate.sh\")) | $1"; }
trigger() { rendered ".steps[] | select(.trigger != null) | $1"; }
dra_prep_plugin() {
  local ref
  ref=$(plugin_ref "$DRA_PIPELINE" "elastic/dra-prep")
  prep ".plugins[] | select(has(\"${ref}\")) | .[\"${ref}\"] | $1"
}

@test "rendered snapshot pipeline is valid YAML with prep, annotate and trigger steps" {
  render_workflow snapshot
  run yq -e '.' "$PIPELINE_OUTPUT"
  [ "$status" -eq 0 ]
  [ "$(rendered '.steps | length')" -eq 3 ]
  [ "$(rendered '.steps[0].command')" = ".buildkite/scripts/stage-dra-artifacts.sh" ]
  [ "$(rendered '.steps[1].command')" = ".buildkite/scripts/dra-annotate.sh snapshot" ]
  [ "$(rendered '.steps[2].trigger')" = "unified-release-dra-processing" ]
}

@test "the upload goes through buildkite-agent pipeline upload of dra-prep-pipeline.yml" {
  render_workflow snapshot
  grep -qx "pipeline upload ${DRA_PIPELINE}" "$BUILDKITE_AGENT_LOG"
}

@test "no unrendered interpolation is left in either workflow" {
  for wf in snapshot staging; do
    render_workflow "$wf"
    [ -z "$(rendered '[.. | select(tag == "!!str") | select(test("\$\{"))] | .[]')" ]
    run grep -q '\${' "$PIPELINE_OUTPUT"
    [ "$status" -eq 1 ]
  done
}

@test "prep step renders key, command and env for snapshot" {
  render_workflow snapshot
  [ "$(prep '.key')" = "dra-prep-snapshot" ]
  [ "$(prep '.label')" = ":package: DRA Prep (snapshot)" ]
  [ "$(prep '.env.DRA_WORKFLOW')" = "snapshot" ]
  [ "$(prep '.env.VERSION_QUALIFIER')" = "" ]
  [ "$(prep '.env.VERSION_QUALIFIER | tag')" = "!!str" ]
}

@test "prep step renders key and env for staging" {
  render_workflow staging
  [ "$(prep '.key')" = "dra-prep-staging" ]
  [ "$(prep '.env.DRA_WORKFLOW')" = "staging" ]
}

@test "prep step carries oblt-google-auth and dra-prep plugins" {
  render_workflow snapshot
  local auth_ref
  auth_ref=$(plugin_ref "$DRA_PIPELINE" "elastic/oblt-google-auth")
  [ -n "$auth_ref" ]
  [ "$(prep ".plugins[] | select(has(\"${auth_ref}\")) | .[\"${auth_ref}\"].project-id")" = "elastic-observability-ci" ]
  [ "$(dra_prep_plugin '.product_id')" = "beats" ]
  [ "$(dra_prep_plugin '.stack_version')" = "9.9.0" ]
  [ "$(dra_prep_plugin '.workflow')" = "snapshot" ]
}

@test "plugins are pinned to semver tags" {
  local ref
  for name in elastic/oblt-google-auth elastic/dra-prep; do
    ref=$(plugin_ref "$DRA_PIPELINE" "$name")
    [[ "$ref" =~ ^${name}#v[0-9]+\.[0-9]+\.[0-9]+$ ]]
  done
}

@test "oblt-google-auth version matches the google_oidc_plugin anchor" {
  local anchor_ref
  anchor_ref=$(yq '.common[] | select(has("google_oidc_plugin")) | .google_oidc_plugin | keys | .[0]' "$PACKAGING_PIPELINE")
  [[ "$anchor_ref" == elastic/oblt-google-auth#* ]]
  [ "$(plugin_ref "$DRA_PIPELINE" "elastic/oblt-google-auth")" = "$anchor_ref" ]
}

@test "annotate and trigger depend on the prep step" {
  for wf in snapshot staging; do
    render_workflow "$wf"
    local prep_key
    prep_key=$(prep '.key')
    [ "$prep_key" = "dra-prep-${wf}" ]
    [ "$(annotate '.depends_on')" = "$prep_key" ]
    [ "$(trigger '.depends_on')" = "$prep_key" ]
  done
}

@test "annotate passes the workflow" {
  render_workflow staging
  [ "$(annotate '.command')" = ".buildkite/scripts/dra-annotate.sh staging" ]
}

@test "trigger targets unified-release-dra-processing with DRA env" {
  render_workflow snapshot
  [ "$(trigger '.trigger')" = "unified-release-dra-processing" ]
  [ "$(trigger '.build.env.DRA_PRODUCT_ID')" = "beats" ]
  [ "$(trigger '.build.env.DRA_STACK_VERSION')" = "9.9.0" ]
  [ "$(trigger '.build.env.DRA_WORKFLOW')" = "snapshot" ]
}

@test "step keys are unique across snapshot and staging renders" {
  render_workflow snapshot
  cp "$PIPELINE_OUTPUT" "$TEST_TMPDIR/snapshot.yml"
  render_workflow staging
  cp "$PIPELINE_OUTPUT" "$TEST_TMPDIR/staging.yml"
  local keys
  keys=$(yq -N '.steps[] | select(has("key")) | .key' "$TEST_TMPDIR/snapshot.yml" "$TEST_TMPDIR/staging.yml")
  [ "$(printf '%s\n' "$keys" | wc -l)" -eq 2 ]
  [ -z "$(printf '%s\n' "$keys" | sort | uniq -d)" ]
  # Parent-build keys must not collide with the sub-pipeline's keys either.
  local parent_keys
  parent_keys=$(yq 'explode(.) | .. | select(tag == "!!map") | select(has("key")) | .key' "$PACKAGING_PIPELINE")
  [ -z "$(printf '%s\n%s\n' "$keys" "$parent_keys" | sort | uniq -d)" ]
}

@test "staging with external VERSION_QUALIFIER embeds it in the stack version" {
  export VERSION_QUALIFIER="alpha1"
  render_workflow staging
  [ "$(dra_prep_plugin '.stack_version')" = "9.9.0-alpha1" ]
  [ "$(trigger '.build.env.DRA_STACK_VERSION')" = "9.9.0-alpha1" ]
  [ "$(prep '.env.VERSION_QUALIFIER')" = "alpha1" ]
  [ ! -s "$CURL_LOG" ]
}

@test "staging with bucket qualifier embeds it in the stack version" {
  export MOCK_GCS_QUALIFIER="beta1"
  render_workflow staging
  [ "$(dra_prep_plugin '.stack_version')" = "9.9.0-beta1" ]
  [ "$(trigger '.build.env.DRA_STACK_VERSION')" = "9.9.0-beta1" ]
  [ "$(prep '.env.VERSION_QUALIFIER')" = "beta1" ]
  grep -q "dra-qualifier/9.9" "$CURL_LOG"
}

@test "staging without qualifier uses the plain version" {
  render_workflow staging
  [ "$(dra_prep_plugin '.stack_version')" = "9.9.0" ]
  [ "$(trigger '.build.env.DRA_STACK_VERSION')" = "9.9.0" ]
}

@test "snapshot ignores the qualifier bucket" {
  export MOCK_GCS_QUALIFIER="beta1"
  render_workflow snapshot
  [ "$(dra_prep_plugin '.stack_version')" = "9.9.0" ]
  [ "$(trigger '.build.env.DRA_STACK_VERSION')" = "9.9.0" ]
  [ ! -s "$CURL_LOG" ]
}

@test "DRY_RUN unset renders upload as true" {
  render_workflow snapshot
  [ "$(dra_prep_plugin '.upload')" = "true" ]
  # Buildkite interpolates after parsing, so the value reaches the plugin as
  # a string, not a YAML boolean.
  [ "$(dra_prep_plugin '.upload | tag')" = "!!str" ]
}

@test "DRY_RUN=true renders upload as false" {
  export DRY_RUN="true"
  render_workflow snapshot
  [ "$(dra_prep_plugin '.upload')" = "false" ]
  # Buildkite interpolates after parsing, so the value reaches the plugin as
  # a string, not a YAML boolean.
  [ "$(dra_prep_plugin '.upload | tag')" = "!!str" ]
}

@test "annotate and trigger are skipped when DRY_RUN is true" {
  export DRY_RUN="true"
  render_workflow staging
  local expected="build.env('DRY_RUN') != \"true\""
  [ "$(annotate '.if')" = "$expected" ]
  [ "$(trigger '.if')" = "$expected" ]
  [ "$(prep '.if')" = "null" ]
}

@test "packaging pipeline DRA group has dra-snapshot and dra-staging" {
  [ "$(yq '.steps[] | select(.group == "DRA publish") | .key' "$PACKAGING_PIPELINE")" = "dra" ]
  [ "$(dra_step dra-snapshot .key)" = "dra-snapshot" ]
  [ "$(dra_step dra-staging .key)" = "dra-staging" ]
  [ "$(dra_step dra-snapshot .env.WORKFLOW)" = "snapshot" ]
  [ "$(dra_step dra-staging .env.WORKFLOW)" = "staging" ]
}

@test "packaging pipeline DRA steps keep their if conditions" {
  [ "$(dra_step dra-snapshot .if)" = "(build.branch =~ /^[0-9]+\.[0-9x]+\\\$/ || build.branch == 'main' || build.env('RUN_SNAPSHOT') == \"true\") && build.env('VERSION_QUALIFIER') == null" ]
  [ "$(dra_step dra-staging .if)" = "(build.branch =~ /^[0-9]+\.[0-9x]+\\\$/ || build.env('VERSION_QUALIFIER') != null) || build.env('RUN_STAGING') == \"true\"" ]
}

@test "packaging pipeline DRA steps keep their depends_on lists" {
  for wf in snapshot staging; do
    [ "$(dra_step "dra-${wf}" '.depends_on | join(",")')" = "start-gate-${wf},packaging-${wf},dashboards-${wf}" ]
  done
}

@test "packaging pipeline DRA steps upload dra-prep-pipeline.yml after resolving the version" {
  for wf in snapshot staging; do
    local cmd
    cmd=$(dra_step "dra-${wf}" .command)
    [ "$(printf '%s\n' "$cmd" | sed -n 1p)" = "source .buildkite/scripts/packaging/resolve-dra-version.sh" ]
    [ "$(printf '%s\n' "$cmd" | sed -n 2p)" = "buildkite-agent pipeline upload ${DRA_PIPELINE}" ]
  done
}
