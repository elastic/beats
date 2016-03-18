#!/bin/bash
set -e

# Script directory:
SDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

usage() {
cat << EOF
  Usage: $(basename $0) [-vh] [-d=directory] [-b=bucket] [-p=prefix]
    [--access-key-id=aws id] [--secret-key-id=aws secret]

  Description: Sign packages and publish them to APT and YUM repositories
    hosted from an S3 bucket. When publishing, the repository metadata is
    also signed to prevent tampering.

    You will be prompted once for the GPG signing key's password. If the
    PASS environment variable is set then that value will be used and you
    will not be prompted.

  Options:
    --aws-access-key=AWS_ACCESS_KEY  Required. AWS access key. Alternatively,
                                     AWS_ACCESS_KEY may be set as an environment
                                     variable.

    --aws-secret-key=AWS_SECRET_KEY  Required. AWS secret key. Alternatively,
                                     AWS_SECRET_KEY may be set as an environment
                                     variable.

    -b=BUCKET | --bucket=BUCKET      Required. The S3 bucket in which to publish.

    -p=PREFIX | --prefix=PREFIX      Required. Path to prefix to all published
                                     repositories.

    -d=DIR | --directory=DIR         Required. Directory to recursively search
                                     for .rpm and .deb files.

    -g=GPG_KEY | --gpg-key=GPG_KEY   Optional. Path to GPG key file to import.

    -o=ORIGIN | --origin=ORIGIN      Optional. Origin to use in APT repo metadata.

    -v | --verbose                   Optional. Enable verbose logging to stderr.
    
    -h | --help                      Optional. Print this usage information.
EOF
}

# Write a debug message to stderr.
debug()
{
  if [ "$VERBOSE" == "true" ]; then
    echo "DEBUG: $1" >&2
  fi
}

# Write and error message to stderr.
err()
{
  echo "ERROR: $1" >&2
}

# Parse command line arguments.
parseArgs() {
  for i in "$@"
  do
  case $i in
    --aws-access-key=*)
      AWS_ACCESS_KEY="${i#*=}"
      shift
      ;;
    --aws-secret-key=*)
      AWS_SECRET_KEY="${i#*=}"
      shift
      ;;
    -b=*|--bucket=*)
      BUCKET="${i#*=}"
      shift
      ;;
    -d=*|--directory=*)
      DIRECTORY="${i#*=}"
      shift
      ;;
    -g=*|--gpg-key=*)
      GPG_KEY="${i#*=}"
      shift
      ;;
    -h|--help)
      usage
      exit 1
      ;;
    -o=*|--origin=*)
      ORIGIN="${i#*=}"
      shift
      ;;
    -p=*|--prefix=*)
      PREFIX="${i#*=}"
      shift
      ;;
    -v|--verbose)
      VERBOSE=true
      shift
      ;;
    *)
      echo "Invalid argument: $i"
      usage
      exit 1
      ;;
  esac
  done

  if [ -z "$BUCKET" ]; then
    err "-b=BUCKET or --bucket=BUCKET is required."
    exit 1
  fi

  if [ -z "$DIRECTORY" ]; then
    err "-d=DIRECTORY or --directory=DIRECTORY is required."
    exit 1
  fi
  
  if [ ! -e "$DIRECTORY" ]; then
    err "Directory $DIRECTORY does not exists."
    exit 1
  fi

  if [ -z "$PREFIX" ]; then
    err "-p=PREFIX or --prefix=PREFIX is required."
    exit 1
  fi

  if [ -z "$AWS_ACCESS_KEY" ]; then
    err "--access-key-id=AWS_ACCESS_KEY is required."
    exit 1
  fi

  if [ -z "$AWS_SECRET_KEY" ]; then
    err "--secret-access-key-id=AWS_SECRET_KEY is required."
    exit 1
  fi

  export BUCKET
  export ORIGIN 
  export PREFIX
  export AWS_ACCESS_KEY
  export AWS_SECRET_KEY
}

importGpg() {
  if [ ! -z "$GPG_KEY" ]; then
    if [ ! -f "$GPG_KEY" ]; then
      err "GPG key file $GPG_KEY does not exists."
      exit 1
    fi

    debug "Importing GPG key $GPG_KEY"
    gpg --import --allow-secret-key-import "$GPG_KEY" | true
  else
    debug "Not importing a GPG key because --gpg-key not specified."
  fi
}

getPassword() {
  if [ -z "$PASS" ]; then
    echo -n "Enter GPG pass phrase: "
    read -s PASS
  fi

  export PASS
}

signDebianPackages() {
  debug "Entering signDebianPackages"
  find $DIRECTORY -name '*.deb' | xargs expect $SDIR/debsign.expect
  debug "Exiting signDebianPackages"
}

signRpmPackages() {
  debug "Entering signRpmPackages"
  find $DIRECTORY -name '*.rpm' | xargs expect $SDIR/rpmsign.expect
  debug "Exiting signRpmPackages"
}

publishToAptRepo() {
  debug "Entering publishToAptRepo"

  # Verify the repository and credentials before continuing.
  deb-s3 verify --bucket "$BUCKET" --prefix "${PREFIX}/apt"

  for arch in i386 amd64
  do
    debug "Publishing $arch .deb packages..."
    export arch

    for deb in $(find "$DIRECTORY" -name "*${arch}.deb")
    do
      expect $SDIR/deb-s3.expect "$deb"
    done
  done
}

publishToYumRepo() {
  debug "Entering publishToYumRepo"

  for arch in i686 x86_64
  do
    debug "Publishing $arch .rpm packages..."
    export arch

    for rpm in $(find "$DIRECTORY" -name "*${arch}.rpm")
    do
      expect $SDIR/rpm-s3.expect "$rpm"
    done
  done
}

main() {
  parseArgs $*
  importGpg
  getPassword
  signDebianPackages
  signRpmPackages
  publishToAptRepo
  publishToYumRepo
  debug "Success"
}

main $*
