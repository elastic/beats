#!/usr/bin/env bash

set -euo pipefail

add_bin_path() {
    echo "Adding PATH to the environment variables..."
    create_bin
    export PATH="${PATH}:${BIN}"
}

with_go() {
    local go_version="${GOLANG_VERSION}"
    echo "Setting up the Go environment..."
    create_bin
    check_platform_architecture
    retry 5 curl -sL -o ${BIN}/gvm "https://github.com/andrewkroh/gvm/releases/download/${SETUP_GVM_VERSION}/gvm-${PLATFORM_TYPE}-${arch_type}"
    export PATH="${PATH}:${BIN}"
    chmod +x ${BIN}/gvm
    eval "$(gvm "$go_version")"
    go version
    which go
    export PATH="${PATH}:$(go env GOPATH):$(go env GOPATH)/bin"
}

with_mage() {
    local install_packages=(
            "github.com/magefile/mage"
            "github.com/elastic/go-licenser"
            "golang.org/x/tools/cmd/goimports"
            "github.com/jstemmer/go-junit-report"
            "gotest.tools/gotestsum"
    )
    create_bin
    for pkg in "${install_packages[@]}"; do
        go install "${pkg}@latest"
    done
}

create_bin() {
    if [[ ! -d "${BIN}" ]]; then
    mkdir -p ${BIN}
    fi
}

check_platform_architecture() {
# for downloading the GVM and Terraform packages
  case "${HW_TYPE}" in
   "x86_64")
        arch_type="amd64"
        ;;
    "aarch64")
        arch_type="arm64"
        ;;
    "arm64")
        arch_type="arm64"
        ;;
    *)
    echo "The current platform/OS type is unsupported yet"
    ;;
  esac
}

retry() {
    local retries=$1
    shift
    local count=0
    until "$@"; do
        exit=$?
        wait=$((2 ** count))
        count=$((count + 1))
        if [ $count -lt "$retries" ]; then
            >&2 echo "Retry $count/$retries exited $exit, retrying in $wait seconds..."
            sleep $wait
        else
            >&2 echo "Retry $count/$retries exited $exit, no more retries left."
            return $exit
        fi
    done
    return 0
}

are_files_changed() {
  local changeset=$1

  if git diff --name-only HEAD@{1} HEAD | grep -qE "$changeset"; then
    return 0;
  else
    return 1;
  fi
}

changeset_applies() {
  local changeset=$1
  if are_files_changed "$changeset"; then
    echo true
  else
    echo false
  fi
}
