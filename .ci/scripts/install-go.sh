#!/usr/bin/env bash
set -exuo pipefail

MSG="parameter missing."
GO_VERSION=${GO_VERSION:?$MSG}
PROPERTIES_FILE=${PROPERTIES_FILE:-"go_env.properties"}
HOME=${HOME:?$MSG}
GOOS=$(uname -s| tr '[:upper:]' '[:lower:]')
GVM_CMD="${HOME}/bin/gvm"

case $(uname -m) in
  x86_64|amd64)
    GOARCH=amd64
    ;;
  i686|i386)
    GOARCH=386
    ;;
  aarch64)
    GOARCH=arm64
    ;;
  armv7l)
    GOARCH=arm
    ;;
  *)
    GOARCH=$(uname -m)
    ;;
esac

mkdir -p "${HOME}/bin"

curl -sSLo "${GVM_CMD}" "https://github.com/andrewkroh/gvm/releases/download/v0.2.2/gvm-${GOOS}-${GOARCH}"
chmod +x "${GVM_CMD}"

gvm ${GO_VERSION}|cut -d ' ' -f 2|tr -d '\"' > ${PROPERTIES_FILE}

eval $(gvm ${GO_VERSION})
