#!/bin/bash
#
# Helper script to check that packages defined in a requirements.txt
# file can be installed in different Python versions, it checks by
# default the requirements.txt file for libbeat tests.
#
#    Usage: check_python_requirements.sh /path/to/requirements.txt
#
# VERSIONS environment variable can be set to a space-separated list
# of versions of python to test with.
#

set -e

function abspath() {
	local path=$1
	if [ -d "$path" ]; then
		cd "$path"; pwd; cd - > /dev/null
	else
		echo $(abspath "$(dirname "$path")")/$(basename "$path")
	fi
}

BEATS_PATH=$(abspath "$(dirname "${BASH_SOURCE[0]}")"/..)

VERSIONS=${VERSIONS:-3.5 3.6 3.7 3.8 3.9-rc}
REQUIREMENTS=${1:-${BEATS_PATH}/libbeat/tests/system/requirements.txt}

if [ ! -f "$REQUIREMENTS" ]; then
	echo "Requirements file doesn't exist: $REQUIREMENTS"
	exit -1
fi

REQUIREMENTS=$(abspath "$REQUIREMENTS")

echo "Versions: $VERSIONS"
echo "Requirements file: $REQUIREMENTS"

for version in $VERSIONS; do
	echo "==== Version: $version"

	docker run -it --rm -v "$REQUIREMENTS":/requirements.txt python:$version \
		python -m pip install -q -r /requirements.txt

	echo "==== OK"
done
