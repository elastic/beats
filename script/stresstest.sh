#!/bin/bash
#
# stresstest.sh is a helper script to run specific Go tests with x/tools/cmd/stress.

set -e

function usage() {
  echo "Usage: $0 [--tags tag1,tag2,...] [package_path] TestName [stress options...]"
  echo ""
  echo "--tags: Optional comma-separated list of build tags"
  echo "package_path: Path to the Go package containing the tests."
  echo "TestName: Regular expression to match the test to run, equivalent to -test.run."
  echo "[stress options]: Options to pass to the stress command."
  echo ""
  echo "Examples:"
  echo "  $0 ./libbeat/common/backoff ^TestBackoff$ -p 32"
  echo "  $0 --tags integration ./libbeat/common/backoff ^TestBackoff$ -p 32"
}

if [[ "$1" == "--help" || "$1" == "-h" ]]; then
  usage
  exit 0
fi

# Parse optional --tags parameter
build_tags=""
if [[ "$1" == "--tags" ]]; then
  if [[ $# -lt 2 ]]; then
    echo "Error: --tags requires a value."
    usage
    exit 1
  fi
  build_tags="$2"
  shift 2
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
test_exec_file="$(basename "$test_package_path").test"
test_regex=${2}
stress_options=("${@:3}")

cd "$test_package_path"
rm "$test_exec_file" 2>/dev/null || true
if [[ -n "$build_tags" ]]; then
  go test -tags "$build_tags" -c -o "./$test_exec_file"
else
  go test -c -o "./$test_exec_file"
fi
trap 'rm "./$test_exec_file" 2>/dev/null || true' EXIT INT TERM
go run golang.org/x/tools/cmd/stress@latest "${stress_options[@]}" "./$test_exec_file" -test.run "$test_regex" -test.v
