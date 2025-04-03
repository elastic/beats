#!/bin/bash
junitfile=$1 # filename for jnit annotation plugin

set -euo pipefail

echo "--- Pre install"
source .buildkite/scripts/pre-install-command.sh
go version
add_bin_path
with_go_junit_report

echo "--- Go Test fips140=only"
set +e
GODEBUG=fips140=only go test -tags=integration,requirefips -json -race -v ./... > test-fips-report.json
exit_code=$?
set -e

# Create Junit report for junit annotation plugin
go-junit-report -parser gojson > "${junitfile:-junit-report-fips-linux.xml}" < test-fips-report.json
exit $exit_code
