#!/bin/bash
#
# stresstest.sh is a helper script to run specific Go tests with x/tools/cmd/stress.

set -e

function usage() {
  echo "Usage: $0 [package_path] TestName [stress options...]"
  echo ""
  echo "package_path: Path to the Go package containing the tests."
  echo "TestName: Regular expression to match the test to run, equivalent to -test.run."
  echo "[stress options]: Options to pass to the stress command."
  echo ""
  echo "Example: $0 ./libbeat/common/backoff ^TestBackoff$ -p 32"
}

if [[ "$1" == "--help" || "$1" == "-h" ]]; then
  usage
  exit 0
fi

if [[ $# -lt 2 ]]; then
  echo "Error: Missing arguments."
  usage
  exit 1
fi

if [ ! -d "$1" ]; then
  echo "Error: Package path '$1' does not exist."
  usage
  exit 1
fi

test_package_path=${1}
test_exec_file="$(basename $test_package_path).test"
test_regex=${2}
stress_options=("${@:3}")

cd "$test_package_path"
rm "$test_exec_file" 2>/dev/null || true
go test -c -o "./$test_exec_file"
trap 'rm "./$test_exec_file" 2>/dev/null || true' EXIT INT TERM
go run golang.org/x/tools/cmd/stress@latest "${stress_options[@]}" "./$test_exec_file" -test.run "$test_regex" -test.v
