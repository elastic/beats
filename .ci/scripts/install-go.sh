#!/usr/bin/env bash
set -exuo pipefail

MSG="environment variable missing"
GO_VERSION=${GO_VERSION:?$MSG}
PROPERTIES_FILE=${PROPERTIES_FILE:-"go_env.properties"}
HOME=${HOME:?$MSG}
ARCH=$(uname -s| tr '[:upper:]' '[:lower:]')
GVM_CMD="${HOME}/bin/gvm"

if command -v go
then
    echo "Found Go. Checking version.."
    FOUND_GO_VERSION=$(go version|awk '{print $3}'|sed s/go//)
    if [ $FOUND_GO_VERSION == $GO_VERSION ]
    then
        echo "Versions match. No need to install Go. Exiting."
        exit 0
    fi
fi

echo "UNMET DEP: Installing Go"
mkdir -p "${HOME}/bin"

curl -sSLo "${GVM_CMD}" "https://github.com/andrewkroh/gvm/releases/download/v0.2.2/gvm-${ARCH}-amd64"
chmod +x "${GVM_CMD}"

gvm ${GO_VERSION}|cut -d ' ' -f 2|tr -d '\"' > ${PROPERTIES_FILE}

eval $(gvm ${GO_VERSION})
