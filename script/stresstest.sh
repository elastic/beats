#!/bin/bash
#
# stresstest.sh is a helper script to run specific Go tests with x/tools/cmd/stress.
#
# Usage: ./script/stresstest.sh [package_path] ^TestName$ [stress options...]
#
# Example: ./script/stresstest.sh ./libbeat/common/backoff ^TestBackoff$ -p 32

set -e

test_package_path=${1:-$(pwd)}
test_exec_file="$(basename $test_package_path).test"
test_regex=${2}
stress_options=("${@:3}")

cd "$test_package_path"
rm "$test_exec_file" 2>/dev/null || true
go test -c -o "./$test_exec_file"
trap 'rm "./$test_exec_file" 2>/dev/null || true' EXIT INT TERM
go run golang.org/x/tools/cmd/stress@latest "${stress_options[@]}" "./$test_exec_file" -test.run "$test_regex" -test.v
