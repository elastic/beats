#!/bin/sh

function test_module() {
  local module=$1

  echo ">>> Checking Docker images for $module"
  MODULE=${module} /tmp/beats/.ci/metricbeat/bats-core/bin/bats --tap tests | tee /tmp/beats/.ci/metricbeat/target/results.tap
}

function main() {
  test_module "${@}"
}

main "${@}"