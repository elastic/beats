#!/usr/bin/env bash
set -euo pipefail

WORKSPACE=${WORKSPACE:-"$(pwd)"}
BIN="${WORKSPACE}/bin"
platform_type="$(uname)"
platform_type_lowercase=$(echo "$platform_type" | tr '[:upper:]' '[:lower:]')
arch_type="$(uname -m)"
GITHUB_PR_TRIGGER_COMMENT=${GITHUB_PR_TRIGGER_COMMENT:-""}
GITHUB_PR_LABELS=${GITHUB_PR_LABELS:-""}
ONLY_DOCS=${ONLY_DOCS:-"true"}

[ -z "${run_libbeat+x}" ] && run_libbeat="$(buildkite-agent meta-data get run_libbeat --default "false")"
[ -z "${run_metricbeat+x}" ] && run_metricbeat="$(buildkite-agent meta-data get run_metricbeat --default "false")"
[ -z "${run_packetbeat+x}" ] && run_packetbeat="$(buildkite-agent meta-data get run_packetbeat --default "false")"
[ -z "${run_winlogbeat+x}" ] && run_winlogbeat="$(buildkite-agent meta-data get run_winlogbeat --default "false")"
[ -z "${run_libbeat_arm_test+x}" ] && run_libbeat_arm_test="$(buildkite-agent meta-data get run_libbeat_arm_test --default "false")"
[ -z "${run_packetbeat_arm_test+x}" ] && run_packetbeat_arm_test="$(buildkite-agent meta-data get run_packetbeat_arm_test --default "false")"
[ -z "${run_metricbeat_macos_tests+x}" ] && run_metricbeat_macos_tests="$(buildkite-agent meta-data get run_metricbeat_macos_tests --default "false")"
[ -z "${run_packetbeat_macos_tests+x}" ] && run_packetbeat_macos_tests="$(buildkite-agent meta-data get run_packetbeat_macos_tests --default "false")"
trigger_specific_beat="run_${BEATS_PROJECT_NAME}"
trigger_specific_arm_tests="run_${BEATS_PROJECT_NAME}_arm_tests"
trigger_specific_macos_tests="run_${BEATS_PROJECT_NAME}_macos_tests"

metricbeat_changeset=(
  "^metricbeat/.*"
  )

libbeat_changeset=(
  "^libbeat/.*"
  )

packetbeat_changeset=(
  "^packetbeat/.*"
  )

winlogbeat_changeset=(
  "^winlogbeat/.*"
  )

oss_changeset=(
  "^go.mod"
  "^pytest.ini"
  "^dev-tools/.*"
  "^libbeat/.*"
  "^testing/.*"
)

ci_changeset=(
  "^.buildkite/.*"
)

go_mod_changeset=(
  "^go.mod"
  )

docs_changeset=(
  ".*\\.(asciidoc|md)"
  "deploy/kubernetes/.*-kubernetes\\.yaml"
  )

packaging_changeset=(
  "^dev-tools/packaging/.*"
  ".go-version"
  )


with_docker_compose() {
  local version=$1
  echo "Setting up the Docker-compose environment..."
  create_workspace
  retry 3 curl -sSL -o ${BIN}/docker-compose "https://github.com/docker/compose/releases/download/${version}/docker-compose-${platform_type_lowercase}-${arch_type}"
  chmod +x ${BIN}/docker-compose
  export PATH="${BIN}:${PATH}"
  docker-compose version
}

create_workspace() {
  if [[ ! -d "${BIN}" ]]; then
    mkdir -p "${BIN}"
  fi
}

add_bin_path() {
  echo "Adding PATH to the environment variables..."
  create_workspace
  export PATH="${BIN}:${PATH}"
}

check_platform_architeture() {
  case "${arch_type}" in
    "x86_64")
      go_arch_type="amd64"
      ;;
    "aarch64")
      go_arch_type="arm64"
      ;;
    "arm64")
      go_arch_type="arm64"
      ;;
    *)
    echo "The current platform or OS type is unsupported yet"
    ;;
  esac
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

with_go() {
  echo "Setting up the Go environment..."
  create_workspace
  check_platform_architeture
  retry 5 curl -sL -o "${BIN}/gvm" "https://github.com/andrewkroh/gvm/releases/download/${SETUP_GVM_VERSION}/gvm-${platform_type_lowercase}-${go_arch_type}"
  chmod +x "${BIN}/gvm"
  eval "$(gvm $GO_VERSION)"
  go version
  which go
  local go_path="$(go env GOPATH):$(go env GOPATH)/bin"
  export PATH="${go_path}:${PATH}"
}

checkLinuxType() {
  if [ "${platform_type}" == "Linux" ]; then
    if grep -q 'ubuntu' /etc/os-release; then
      echo "ubuntu"
    elif grep -q 'rhel' /etc/os-release; then
      echo "rhel"
    else
      echo "Unsupported Linux"
    fi
  else
      echo "This is not a Linux"
  fi
}

with_python() {
  local linuxType="$(checkLinuxType)"
  echo "${linuxType}"
  if [ "${platform_type}" == "Linux" ]; then
    if [ "${linuxType}" = "ubuntu" ]; then
      sudo apt-get update
      sudo apt-get install -y python3-pip python3-venv
    elif [ "${linuxType}" = "rhel" ]; then
      sudo dnf update -y
      sudo dnf install -y python3 python3-pip
      pip3 install virtualenv
    fi
  elif [ "${platform_type}" == "Darwin" ]; then
    brew update
    pip3 install virtualenv
    ulimit -Sn 10000
  fi
}

with_dependencies() {
  local linuxType="$(checkLinuxType)"
  echo "${linuxType}"
  if [ "${platform_type}" == "Linux" ]; then
    if [ "${linuxType}" = "ubuntu" ]; then
      sudo apt-get update
      sudo apt-get install -y libsystemd-dev libpcap-dev
    elif [ "${linuxType}" = "rhel" ]; then
      # sudo dnf update -y
      sudo dnf install -y systemd-devel
      wget https://mirror.stream.centos.org/9-stream/CRB/${arch_type}/os/Packages/libpcap-devel-1.10.0-4.el9.${arch_type}.rpm     #TODO: move this step to our own image
      sudo dnf install -y libpcap-devel-1.10.0-4.el9.${arch_type}.rpm     #TODO: move this step to our own image
    fi
  elif [ "${platform_type}" == "Darwin" ]; then
    pip3 install libpcap
  fi
}

config_git() {
  if [ -z "$(git config --get user.email)" ]; then
    git config --global user.email "beatsmachine@users.noreply.github.com"
    git config --global user.name "beatsmachine"
  fi
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

are_paths_changed() {
  local patterns=("${@}")
  local changelist=()
  for pattern in "${patterns[@]}"; do
    changed_files=($(git diff --name-only HEAD@{1} HEAD | grep -E "$pattern"))
    if [ "${#changed_files[@]}" -gt 0 ]; then
      changelist+=("${changed_files[@]}")
    fi
  done

  if [ "${#changelist[@]}" -gt 0 ]; then
    echo "Files changed:"
    echo "${changelist[*]}"
    return 0
  else
    echo "No files changed within specified changeset:"
    echo "${patterns[*]}"
    return 1
  fi
}

are_changed_only_paths() {
  local patterns=("${@}")
  local changelist=()
  local changed_files=$(git diff --name-only HEAD@{1} HEAD)
  if [ -z "$changed_files" ] || grep -qE "$(IFS=\|; echo "${patterns[*]}")" <<< "$changed_files"; then
    return 0
  fi
  return 1
}

are_conditions_met_mandatory_tests() {
  declare -n beat_changeset_reference="${BEATS_PROJECT_NAME}_changeset"
  if are_paths_changed "${oss_changeset[@]}" || are_paths_changed "${ci_changeset[@]}" ]]; then   # from https://github.com/elastic/beats/blob/c5e79a25d05d5bdfa9da4d187fe89523faa42afc/metricbeat/Jenkinsfile.yml#L3-L12
    return 0
  fi
  if are_paths_changed "${beat_changeset_reference[@]}" || [[ "${GITHUB_PR_TRIGGER_COMMENT}" == "/test ${BEATS_PROJECT_NAME}" || "${GITHUB_PR_LABELS}" =~ /(?i)${BEATS_PROJECT_NAME}/ || "${!trigger_specific_beat}" == "true" ]]; then
    return 0
  fi
  return 1
}

are_conditions_met_arm_tests() {
  if are_conditions_met_mandatory_tests; then    #from https://github.com/elastic/beats/blob/c5e79a25d05d5bdfa9da4d187fe89523faa42afc/Jenkinsfile#L145-L171
    if [[ "$BUILDKITE_PIPELINE_SLUG" == "beats-libbeat" || "$BUILDKITE_PIPELINE_SLUG" == "beats-packetbeat" ]]; then
      if [[ "${GITHUB_PR_TRIGGER_COMMENT}" == "/test ${BEATS_PROJECT_NAME} for arm" || "${GITHUB_PR_LABELS}" =~ arm || "${!trigger_specific_arm_tests}" == "true" ]]; then
        return 0
      fi
    fi
  fi
  return 1
}

are_conditions_met_macos_tests() {
  if are_conditions_met_mandatory_tests; then    #from https://github.com/elastic/beats/blob/c5e79a25d05d5bdfa9da4d187fe89523faa42afc/Jenkinsfile#L145-L171
    if [[ "$BUILDKITE_PIPELINE_SLUG" == "beats-metricbeat" || "$BUILDKITE_PIPELINE_SLUG" == "beats-packetbeat" ]]; then
      if [[ "${GITHUB_PR_TRIGGER_COMMENT}" == "/test ${BEATS_PROJECT_NAME} for macos" || "${GITHUB_PR_LABELS}" =~ macOS || "${!trigger_specific_macos_tests}" == "true" ]]; then   # from https://github.com/elastic/beats/blob/c5e79a25d05d5bdfa9da4d187fe89523faa42afc/metricbeat/Jenkinsfile.yml#L3-L12
        return 0
      fi
    fi
  fi
  return 1
}

are_conditions_met_packaging() {
  if are_conditions_met_mandatory_tests; then    #from https://github.com/elastic/beats/blob/c5e79a25d05d5bdfa9da4d187fe89523faa42afc/Jenkinsfile#L145-L171
    if [[ "${BUILDKITE_TAG}" == "" || "${BUILDKITE_PULL_REQUEST}" != "" ]]; then
      return 0
    fi
  fi
  return 1
}

config_git() {
  if [ -z "$(git config --get user.email)" ]; then
    git config --global user.email "beatsmachine@users.noreply.github.com"
    git config --global user.name "beatsmachine"
  fi
}

if ! are_changed_only_paths "${docs_changeset[@]}" ; then
  ONLY_DOCS="false"
  echo "Changes include files outside the docs_changeset vairiabe. ONLY_DOCS=$ONLY_DOCS."
else
  echo "All changes are related to DOCS. ONLY_DOCS=$ONLY_DOCS."
fi

if are_paths_changed "${go_mod_changeset[@]}" ; then
  GO_MOD_CHANGES="true"
fi

if are_paths_changed "${packaging_changeset[@]}" ; then
  PACKAGING_CHANGES="true"
fi
