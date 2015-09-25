#!/bin/bash
#
# Contains the Go tool-chain pure-Go bootstrapper, that as of Go 1.5, initiates
# not only a few pre-built Go cross compilers, but rather bootstraps all of the
# supported platforms from the origin Linux amd64 distribution.
#
# Usage: bootstrap.sh
#
# Needed environment variables:
#   FETCH          - Remote file fetcher and checksum verifier (injected by image)
#   ROOT_DIST      - 64 bit Linux Go binary distribution package
#   ROOT_DIST_SHA1 - 64 bit Linux Go distribution package checksum
set -e

# Download, verify and install the root distribution
$FETCH $ROOT_DIST  $ROOT_DIST_SHA1

tar -C /usr/local -xzf `basename $ROOT_DIST`
rm -f `basename $ROOT_DIST`

export GOROOT=/usr/local/go
export GOROOT_BOOTSTRAP=$GOROOT

# Pre-build all guest distributions based on the root distribution
echo "Bootstrapping linux/386..."
GOOS=linux GOARCH=386 CGO_ENABLED=1 go install std

echo "Bootstrapping linux/arm..."
GOOS=linux GOARCH=arm CGO_ENABLED=1 CC=arm-linux-gnueabi-gcc go install std

echo "Bootstrapping windows/amd64..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go install std

echo "Bootstrapping windows/386..."
GOOS=windows GOARCH=386 CGO_ENABLED=1 CC=i686-w64-mingw32-gcc go install std

echo "Bootstrapping darwin/amd64..."
GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 CC=o64-clang go install std

echo "Bootstrapping darwin/386..."
GOOS=darwin GOARCH=386 CGO_ENABLED=1 CC=o32-clang go install std
