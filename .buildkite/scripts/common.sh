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
OSS_MODULE_PATTERN="^[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*"
XPACK_MODULE_PATTERN="^x-pack\\/[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*"
# define if needed run the whole pipeline for the particular beat
[ -z "${run_filebeat+x}" ] && run_filebeat="$(buildkite-agent meta-data get run_filebeat --default "false")"
[ -z "${run_xpack_metricbeat+x}" ] && run_xpack_metricbeat="$(buildkite-agent meta-data get run_xpack_metricbeat --default "false")"
[ -z "${run_xpack_packetbeat+x}" ] && run_xpack_packetbeat="$(buildkite-agent meta-data get run_xpack_packetbeat --default "false")"

# define if needed run ARM platform-specific tests for the particular beat
[ -z "${run_filebeat_arm_tests+x}" ] && run_filebeat_arm_tests="$(buildkite-agent meta-data get run_filebeat_arm_tests --default "false")"
[ -z "${run_xpack_packetbeat_arm_tests+x}" ] && run_xpack_packetbeat_arm_tests="$(buildkite-agent meta-data get run_xpack_packetbeat_arm_tests --default "false")"
# define if needed run MacOS platform-specific tests for the particular beat
[ -z "${run_xpack_metricbeat_macos_tests+x}" ] && run_xpack_metricbeat_macos_tests="$(buildkite-agent meta-data get run_xpack_metricbeat_macos_tests --default "false")"
[ -z "${run_xpack_packetbeat_macos_tests+x}" ] && run_xpack_packetbeat_macos_tests="$(buildkite-agent meta-data get run_xpack_packetbeat_macos_tests --default "false")"

# define if needed run cloud-specific tests for the particular beat
[ -z "${run_xpack_metricbeat_aws_tests+x}" ] && run_xpack_metricbeat_aws_tests="$(buildkite-agent meta-data get run_xpack_metricbeat_aws_tests --default "false")"

winlogbeat_changeset=(
  "^winlogbeat/.*"
  )

xpack_dockerlogbeat_changeset=(
  "^x-pack/dockerlogbeat/.*"
  )

ci_changeset=(
  "^.buildkite/.*"
  )

go_mod_changeset=(
  "^go.mod"
  )

oss_changeset=(
  "^go.mod"
  "^pytest.ini"
  "^dev-tools/.*"
  "^libbeat/.*"
  "^testing/.*"
)

xpack_changeset=(
  "${oss_changeset[@]}"
)

docs_changeset=(
  ".*\\.(asciidoc|md)"
  "deploy/kubernetes/.*-kubernetes\\.yaml"
  )

packaging_changeset=(
  "^dev-tools/packaging/.*"
  ".go-version"
  )

case "${BUILDKITE_PIPELINE_SLUG}" in
  "beats-xpack-metricbeat")
    BEAT_CHANGESET_REFERENCE=${xpack_metricbeat_changeset[@]}
    ;;
  "beats-xpack-packetbeat")
    BEAT_CHANGESET_REFERENCE=${xpack_packetbeat_changeset[@]}
    ;;
  *)
  echo "~~~ The changeset for the ${BUILDKITE_PIPELINE_SLUG} pipeline hasn't been defined yet."
  ;;
esac

check_and_set_beat_vars() {
  local BEATS_PROJECT_NAME=${BEATS_PROJECT_NAME:=""}
  if [[ "${BEATS_PROJECT_NAME:=""}" == *"x-pack/"* ]]; then
    BEATS_XPACK_PROJECT_NAME=${BEATS_PROJECT_NAME//-/}              #remove -
    BEATS_XPACK_PROJECT_NAME=${BEATS_XPACK_PROJECT_NAME//\//_}      #replace / to _
    BEATS_XPACK_LABEL_PROJECT_NAME=${BEATS_PROJECT_NAME//\//-}      #replace / to - for labels
    BEATS_GH_LABEL=${BEATS_XPACK_LABEL_PROJECT_NAME}
    TRIGGER_SPECIFIC_BEAT="run_${BEATS_XPACK_PROJECT_NAME}"
    TRIGGER_SPECIFIC_ARM_TESTS="run_${BEATS_XPACK_PROJECT_NAME}_arm_tests"
    TRIGGER_SPECIFIC_AWS_TESTS="run_${BEATS_XPACK_PROJECT_NAME}_aws_tests"
    TRIGGER_SPECIFIC_MACOS_TESTS="run_${BEATS_XPACK_PROJECT_NAME}_macos_tests"
    TRIGGER_SPECIFIC_WIN_TESTS="run_${BEATS_XPACK_PROJECT_NAME}_win_tests"
    echo "--- Beats project name is $BEATS_XPACK_PROJECT_NAME"
    mandatory_changeset=(
      "${BEAT_CHANGESET_REFERENCE[@]}"
      "${xpack_changeset[@]}"
      "${ci_changeset[@]}"
    )
  else
    BEATS_GH_LABEL=${BEATS_PROJECT_NAME}
    TRIGGER_SPECIFIC_BEAT="run_${BEATS_PROJECT_NAME}"
    TRIGGER_SPECIFIC_ARM_TESTS="run_${BEATS_PROJECT_NAME}_arm_tests"
    TRIGGER_SPECIFIC_AWS_TESTS="run_${BEATS_PROJECT_NAME}_aws_tests"
    TRIGGER_SPECIFIC_MACOS_TESTS="run_${BEATS_PROJECT_NAME}_macos_tests"
    TRIGGER_SPECIFIC_WIN_TESTS="run_${BEATS_PROJECT_NAME}_win_tests"
    echo "--- Beats project name is $BEATS_PROJECT_NAME"
    mandatory_changeset=(
      "${BEAT_CHANGESET_REFERENCE[@]}"
      "${oss_changeset[@]}"
      "${ci_changeset[@]}"
    )
  fi
  BEATS_GH_COMMENT="/test ${BEATS_PROJECT_NAME}"
  BEATS_GH_MACOS_COMMENT="${BEATS_GH_COMMENT} for macos"
  BEATS_GH_ARM_COMMENT="${BEATS_GH_COMMENT} for arm"
  BEATS_GH_AWS_COMMENT="${BEATS_GH_COMMENT} for aws cloud"
  BEATS_GH_WIN_COMMENT="${BEATS_GH_COMMENT} for windows"
  BEATS_GH_MACOS_LABEL="macOS"
  BEATS_GH_ARM_LABEL="arm"
  BEATS_GH_AWS_LABEL="aws"
  BEATS_GH_WIN_LABEL="windows"
}

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
  echo "Download modules to local cache"
  retry 3 go mod download
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
      sudo apt-get install -y libsystemd-dev libpcap-dev librpm-dev
    elif [ "${linuxType}" = "rhel" ]; then
      # sudo dnf update -y
      sudo dnf install -y systemd-devel rpm-devel
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
  local changed_files=($(git diff --name-only HEAD@{1} HEAD))
  local matched_files=()
  for pattern in "${patterns[@]}"; do
    local matched=($(grep -E "${pattern}" <<< "${changed_files[@]}"))
    if [ "${#matched[@]}" -gt 0 ]; then
      matched_files+=("${matched[@]}")
    fi
  done
  if [ "${#matched_files[@]}" -eq "${#changed_files[@]}" ] || [ "${#changed_files[@]}" -eq 0 ]; then
    return 0
  fi
  return 1
}

are_conditions_met_mandatory_tests() {
  if are_paths_changed "${mandatory_changeset[@]}" || [[ "${GITHUB_PR_TRIGGER_COMMENT}" == "${BEATS_GH_COMMENT}" || "${GITHUB_PR_LABELS}" =~ /(?i)${BEATS_GH_LABEL}/ || "${!TRIGGER_SPECIFIC_BEAT}" == "true" ]]; then
    return 0
  fi
  return 1
}

<<<<<<< HEAD
<<<<<<< HEAD
are_conditions_met_arm_tests() {
  if are_conditions_met_mandatory_tests; then    #from https://github.com/elastic/beats/blob/c5e79a25d05d5bdfa9da4d187fe89523faa42afc/Jenkinsfile#L145-L171
    if [[ "$BUILDKITE_PIPELINE_SLUG" == "beats-libbeat" || "$BUILDKITE_PIPELINE_SLUG" == "beats-packetbeat" ]]; then
      if [[ "${GITHUB_PR_TRIGGER_COMMENT}" == "${BEATS_GH_ARM_COMMENT}" || "${GITHUB_PR_LABELS}" =~ ${BEATS_GH_ARM_LABEL} || "${!TRIGGER_SPECIFIC_ARM_TESTS}" == "true" ]]; then
        return 0
      fi
    fi
  fi
  return 1
}

are_conditions_met_macos_tests() {
  if are_conditions_met_mandatory_tests; then    #from https://github.com/elastic/beats/blob/c5e79a25d05d5bdfa9da4d187fe89523faa42afc/Jenkinsfile#L145-L171
    if [[ "$BUILDKITE_PIPELINE_SLUG" == "beats-xpack-metricbeat" ]] ; then
      if [[ "${GITHUB_PR_TRIGGER_COMMENT}" == "${BEATS_GH_MACOS_COMMENT}" || "${GITHUB_PR_LABELS}" =~ ${BEATS_GH_MACOS_LABEL} || "${!TRIGGER_SPECIFIC_MACOS_TESTS}" == "true" ]]; then   # from https://github.com/elastic/beats/blob/c5e79a25d05d5bdfa9da4d187fe89523faa42afc/metricbeat/Jenkinsfile.yml#L3-L12
        return 0
      fi
    fi
  fi
  return 1
}

are_conditions_met_aws_tests() {
  if are_conditions_met_mandatory_tests; then    #from https://github.com/elastic/beats/blob/c5e79a25d05d5bdfa9da4d187fe89523faa42afc/Jenkinsfile#L145-L171
    if [[ "$BUILDKITE_PIPELINE_SLUG" == "beats-xpack-metricbeat" ]]; then
      if [[ "${GITHUB_PR_TRIGGER_COMMENT}" == "${BEATS_GH_AWS_COMMENT}" || "${GITHUB_PR_LABELS}" =~ ${BEATS_GH_AWS_LABEL} || "${!TRIGGER_SPECIFIC_AWS_TESTS}" == "true" ]]; then   # from https://github.com/elastic/beats/blob/c5e79a25d05d5bdfa9da4d187fe89523faa42afc/metricbeat/Jenkinsfile.yml#L3-L12
        return 0
      fi
    fi
  fi
  return 1
}

are_conditions_met_packaging() {
  if are_conditions_met_mandatory_tests; then
    if [[ "${BUILDKITE_TAG}" == "" || "${BUILDKITE_PULL_REQUEST}" != "false" ]]; then
      return 0
    fi
  fi
  return 1
}

defineModuleFromTheChangeSet() {
  # This method gathers the module name, if required, in order to run the ITs only if the changeset affects a specific module.
  # For such, it's required to look for changes under the module folder and exclude anything else such as asciidoc and png files.
  # This method defines and exports the MODULE variable with a particular module name or '' if changeset doesn't affect a specific module
  local project_path=$1
  local project_path_transformed=$(echo "$project_path" | sed 's/\//\\\//g')
  local project_path_exclussion="((?!^${project_path_transformed}\\/).)*\$"
  local exclude=("^(${project_path_exclussion}|((?!\\/module\\/).)*\$|.*\\.asciidoc|.*\\.png)")

  if [[ "$project_path" == *"x-pack/"* ]]; then
    local pattern=("$XPACK_MODULE_PATTERN")
  else
    local pattern=("$OSS_MODULE_PATTERN")
  fi
  local changed_modules=""
  local module_dirs=$(find "$project_path/module" -mindepth 1 -maxdepth 1 -type d)
  for module_dir in $module_dirs; do
    if are_paths_changed $module_dir && ! are_changed_only_paths "${exclude[@]}"; then
      if [[ -z "$changed_modules" ]]; then
        changed_modules=$(basename "$module_dir")
      else
        changed_modules+=",$(basename "$module_dir")"
      fi
    fi
  done
  if [[ -z "$changed_modules" ]]; then # TODO: remove this condition and uncomment the line below when the issue https://github.com/elastic/ingest-dev/issues/2993 is solved
    if [[ "$BUILDKITE_PIPELINE_SLUG" == "beats-xpack-metricbeat" ]]; then
      export MODULE="aws"
    else
      export MODULE="kubernetes"
    fi
  else
    export MODULE="${changed_modules}"  # TODO: remove this line and uncomment the line below when the issue https://github.com/elastic/ingest-dev/issues/2993 is solved
  # export MODULE="${changed_modules}"     # TODO: uncomment the line when the issue https://github.com/elastic/ingest-dev/issues/2993 is solved
  fi
}

terraformInit() {
  local dir=$1
  echo "Terraform Init on $dir"
  pushd "${dir}" > /dev/null
  terraform init
  popd > /dev/null
}

withAWS() {
  # This method gathers the masked AWS credentials from pre-command hook and sets the right AWS variable names.
  export AWS_ACCESS_KEY_ID=$BEATS_AWS_ACCESS_KEY
  export AWS_SECRET_ACCESS_KEY=$BEATS_AWS_SECRET_KEY
  export TEST_TAGS="${TEST_TAGS:+$TEST_TAGS,}aws"
}

startCloudTestEnv() {
  local dir=$1
  withAWS
  echo "--- Run docker-compose services for emulated cloud env"
  docker-compose -f .ci/jobs/docker-compose.yml up -d                     #TODO: move all docker-compose files from the .ci to .buildkite folder before switching to BK
  with_Terraform
  terraformInit "$dir"
  export TF_VAR_BRANCH=$(echo "${BUILDKITE_BRANCH}" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9-]/-/g')
  export TF_VAR_BUILD_ID="${BUILDKITE_BUILD_ID}"
  export TF_VAR_CREATED_DATE=$(date +%s)
  export TF_VAR_ENVIRONMENT="ci"
  export TF_VAR_REPO="${REPO}"
  pushd "${dir}" > /dev/null
  terraform apply -auto-approve
  popd > /dev/null
}

withNodeJSEnv() {
  # HOME="${WORKSPACE}"
  local version=$1
  # local nvmPath="${HOME}/.nvm/versions/node/${version}/bin"
  echo "Installing nvm"
  curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.1/install.sh | bash
  export NVM_DIR="$HOME/.nvm"
  [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
  echo "Installing the NodeJs version $version"
  nvm install "$version"
  # export PATH="${nvmPath}:${PATH}"
  nvm use "$version"
  node --version
}

installNodeJsDependencies() {
  # Install dependencies to run browsers
  if [ "${platform_type}" == "Linux" ]; then
    sudo apt-get install -y \
      libatk1.0-0 \
      libatk-bridge2.0-0 \
      libcups2 \
      libxkbcommon0 \
      libatspi2.0-0 \
      libxcomposite1 \
      libxdamage1 \
      libxfixes3 \
      libxrandr2 \
      libgbm1 \
      libpango-1.0-0 \
      libcairo2 \
      libasound2
    if [ $? -ne 0 ]; then
      echo "Error: Failed to install dependencies."
      exit 1
    else
      echo "Dependencies installed successfully."
    fi
  elif [ "${platform_type}" == "Darwin" ]; then
    echo "TBD"
  else
    echo "Unsupported platform type."
    exit 1
  fi
}

teardown() {
  # Teardown resources after using them
  echo "---Terraform Cleanup"
  .ci/scripts/terraform-cleanup.sh "${MODULE_DIR}"              #TODO: move all docker-compose files from the .ci to .buildkite folder before switching to BK

  echo "---Docker Compose Cleanup"
  docker-compose -f .ci/jobs/docker-compose.yml down -v         #TODO: move all docker-compose files from the .ci to .buildkite folder before switching to BK
}

unset_secrets () {
  for var in $(printenv | sed 's;=.*;;' | sort); do
    if [[ "$var" == AWS_* || "$var" == BEATS_AWS_* ]]; then
      unset "$var"
    fi
  done
}

if ! are_changed_only_paths "${docs_changeset[@]}" ; then
  export ONLY_DOCS="false"
  echo "Changes include files outside the docs_changeset vairiabe. ONLY_DOCS=$ONLY_DOCS."
else
  echo "All changes are related to DOCS. ONLY_DOCS=$ONLY_DOCS."
fi

if are_paths_changed "${go_mod_changeset[@]}" ; then
  export GO_MOD_CHANGES="true"
fi

if are_paths_changed "${packaging_changeset[@]}" ; then
  export PACKAGING_CHANGES="true"
fi

if [[ "$BUILDKITE_STEP_KEY" == "xpack-metricbeat-pipeline" || "$BUILDKITE_STEP_KEY" == "xpack-dockerlogbeat-pipeline" || "$BUILDKITE_STEP_KEY" == "metricbeat-pipeline" ]]; then
  # Set the MODULE env variable if possible, it should be defined before generating pipeline's steps. It is used in multiple pipelines.
  defineModuleFromTheChangeSet "${BEATS_PROJECT_NAME}"
fi

check_and_set_beat_vars
