#!/bin/bash
set -euo pipefail

WORKSPACE=${WORKSPACE:-"$(pwd)"}

create_workspace() {
  if [[ ! -d "${WORKSPACE}/bin" ]]; then
    mkdir -p "${WORKSPACE}/bin"
  fi
}

add_bin_path() {
  echo "Adding PATH to the environment variables..."
  create_workspace
  export PATH="${PATH}:${WORKSPACE}/bin"
}

with_virtualenv() {
    export PATH=$PATH:/root/.local/bin
    python3 -m pip install --user virtualenv
}

with_yq() {
    pip install yq
}

with_mage() {
    local install_packages=(
            "github.com/magefile/mage"
            "github.com/elastic/go-licenser"
            "golang.org/x/tools/cmd/goimports"
            "github.com/jstemmer/go-junit-report"
            "gotest.tools/gotestsum"
    )
    create_workspace
    for pkg in "${install_packages[@]}"; do
        go install "${pkg}@latest"
    done
}

# with_gh() {
#     # GitHub CLI linux amd64
#     curl -L "https://github.com/cli/cli/releases/download/v${GITHUB_CLI}/gh_${GITHUB_CLI}_linux_amd64.tar.gz" --output ./gh.tar.gz
#     mkdir gh
#     tar -xf gh.tar.gz -C ./gh --strip-components=1
#     chmod +x ./gh/bin/gh
#     export PATH=$PATH:$(pwd)/gh/bin/
#     gh --version
# }

with_go() {
    go_version=$1
    url=$(get_gvm_link "${SETUP_GVM_VERSION}")
    create_workspace
    retry 5 curl -sL -o "${WORKSPACE}/bin/gvm" "${url}"
    chmod +x "${WORKSPACE}/bin/gvm"
    ls -l ${WORKSPACE}/bin/
    eval "$(gvm $go_version)"
    go_path="$(go env GOPATH):$(go env GOPATH)/bin"
    export PATH="${PATH}:${go_path}"
    go version
}

# for gvm link
get_gvm_link() {
    gvm_version=$1
    platform_type="$(uname)"
    platform_type_lowercase=$(echo "$platform_type" | tr '[:upper:]' '[:lower:]')
    arch_type="$(uname -m)"
    [[ ${arch_type} == "aarch64" ]] && arch_type="arm64" # gvm do not have 'aarch64' name for archetecture type
    [[ ${arch_type} == "x86_64" ]] && arch_type="amd64"
    echo "https://github.com/andrewkroh/gvm/releases/download/${gvm_version}/gvm-${platform_type_lowercase}-${arch_type}"
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
