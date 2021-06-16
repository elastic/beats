#!/usr/bin/env bash
#
# Given the Golang release version this script will bump the version.
#
# This script is executed by the automation we are putting in place
# and it requires the git add/commit commands.
#
# Parameters:
#	$1 -> the Golang release version to be bumped. Mandatory.
#
set -euo pipefail
MSG="parameter missing."
GO_RELEASE_VERSION=${1:?$MSG}

OS=$(uname -s| tr '[:upper:]' '[:lower:]')

if [ "${OS}" == "darwin" ] ; then
	SED="sed -i .bck"
else
	SED="sed -i"
fi

echo "Update go version ${GO_RELEASE_VERSION}"
echo "${GO_RELEASE_VERSION}" > .go-version
git add .go-version

find . -maxdepth 3 -name Dockerfile -print0 |
    while IFS= read -r -d '' line; do
        ${SED} -E -e "s#(FROM golang):[0-9]+\.[0-9]+\.[0-9]+#\1:${GO_RELEASE_VERSION}#g" "$line"
        ${SED} -E -e "s#(ARG GO_VERSION)=[0-9]+\.[0-9]+\.[0-9]+#\1=${GO_RELEASE_VERSION}#g" "$line"
        git add "${line}"
    done

${SED} -E -e "s#(:go-version:) [0-9]+\.[0-9]+\.[0-9]+#\1 ${GO_RELEASE_VERSION}#g" libbeat/docs/version.asciidoc
git add libbeat/docs/version.asciidoc

git diff --staged --quiet || git commit -m "[Automation] Update go release version to ${GO_RELEASE_VERSION}"
git --no-pager log -1

echo "You can now push and create a Pull Request"
