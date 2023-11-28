#!/bin/bash
junitfile=$1 # filename for jnit annotation plugin

set -euo pipefail

echo "--- Pre install"
source .buildkite/scripts/pre-install-command.sh
go version
add_bin_path
with_go_junit_report

echo "--- Go Test"
set +e
go test -v ./... > tests-report.txt
exit_code=$?
set -e

# Buildkite collapse logs under --- symbols
# need to change --- to anything else or switch off collapsing (note: not available at the moment of this commit)
awk '{gsub("---", "----"); print }' tests-report.txt

# Create Junit report for junit annotation plugin
go-junit-report > "${junitfile:-junit-report-linux.xml}" < tests-report.txt
exit $exit_code
