#
# File: common.bash
#
# Common bash routines.
#

# Script directory:
_sdir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# debug "msg"
# Write a debug message to stderr.
debug()
{
  if [ "$VERBOSE" == "true" ]; then
    echo "DEBUG: $1" >&2
  fi
}

# err "msg"
# Write and error message to stderr.
err()
{
  echo "ERROR: $1" >&2
}

# get_go_version
# Read the project's Go version and return it in the GO_VERSION variable.
# On failure it will exit.
get_go_version() {
  GO_VERSION=$(cat "${_sdir}/../.go-version")
  if [ -z "$GO_VERSION" ]; then
    err "Failed to detect the project's Go version"
    exit 1
  fi
}

# setup_go_root "version"
# This configures the Go version being used. It sets GOROOT and adds
# GOROOT/bin to the PATH. It uses gimme to download the Go version if
# it does not already exist in the ~/.gimme dir.
setup_go_root() {
  local version=${1}
  export PROPERTIES_FILE=go_env.properties
  GO_VERSION="${version}" .ci/scripts/install-go.sh
  # shellcheck disable=SC1090
  source "${PROPERTIES_FILE}" 2> /dev/null
  debug "$(go version)"
}

# setup_go_path "gopath"
# This sets GOPATH and adds GOPATH/bin to the PATH.
setup_go_path() {
  local gopath="${1}"
  if [ -z "$gopath" ]; then return; fi

  # Setup GOPATH.
  export GOPATH="${gopath}"

  # Add GOPATH to PATH.
  export PATH="${GOPATH}/bin:${PATH}"

  debug "GOPATH=${GOPATH}"
}

jenkins_setup() {
  : "${HOME:?Need to set HOME to a non-empty value.}"
  : "${WORKSPACE:?Need to set WORKSPACE to a non-empty value.}"

  if [ -z ${GO_VERSION:-} ]; then
    get_go_version
  fi

  # Setup Go.
  export GOPATH=${WORKSPACE}
  export PATH=${GOPATH}/bin:${PATH}
  eval "$(gvm ${GO_VERSION})"

  # Workaround for Python virtualenv path being too long.
  export TEMP_PYTHON_ENV=$(mktemp -d)
  export PYTHON_ENV="${TEMP_PYTHON_ENV}/python-env"

  # Write cached magefile binaries to workspace to ensure
  # each run starts from a clean slate.
  export MAGEFILE_CACHE="${WORKSPACE}/.magefile"
}

docker_setup() {
  OS="$(uname)"
  case $OS in
    'Darwin')
      # Start the docker machine VM (ignore error if it's already running).
      docker-machine start default || true
      eval $(docker-machine env default)
      ;;
  esac
}
