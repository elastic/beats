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

${SED} -E -e "s#(go:) \"[0-9]+\.[0-9]+\.[0-9]+\"#\1 \"${GO_RELEASE_VERSION}\"#g" .golangci.yml
git add .golangci.yml

git diff --staged --quiet || git commit -m "[Automation] Update go release version to ${GO_RELEASE_VERSION}"
git --no-pager log -1

echo "You can now push and create a Pull Request"
