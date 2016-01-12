#!/bin/bash
#
# Contains the Go tool-chain bootstrapper, that retrieves all the configured
# distribution packages, extracts the binaries and deletes anything not needed.
#
# Usage: bootstrap.sh
#
# Needed environment variables:
#   FETCH - Remote file fetcher and checksum verifier (injected by image)
#   DIST_LINUX_64,  DIST_LINUX_64_SHA1  - 64 bit Linux Go binaries and checksum
#   DIST_LINUX_32,  DIST_LINUX_32_SHA1  - 32 bit Linux Go binaries and checksum
#   DIST_LINUX_ARM, DIST_LINUX_ARM_SHA1 - ARM v5 Linux Go binaries and checksum
#   DIST_OSX_64,    DIST_OSX_64_SHA1    - 64 bit Mac OSX Go binaries and checksum
#   DIST_OSX_32,    DIST_OSX_32_SHA1    - 32 bit Mac OSX Go binaries and checksum
#   DIST_WIN_64,    DIST_WIN_64_SHA1    - 64 bit Windows Go binaries and checksum
#   DIST_WIN_32,    DIST_WIN_32_SHA1    - 32 bit Windows Go binaries and checksum 
set -e

# Download and verify all the binary packages
$FETCH $DIST_LINUX_64  $DIST_LINUX_64_SHA1
$FETCH $DIST_LINUX_32  $DIST_LINUX_32_SHA1
$FETCH $DIST_OSX_64    $DIST_OSX_64_SHA1
$FETCH $DIST_WIN_64    $DIST_WIN_64_SHA1
$FETCH $DIST_WIN_32    $DIST_WIN_32_SHA1

# Extract the 64 bit Linux package as the primary Go SDK
tar -C /usr/local -xzf `basename $DIST_LINUX_64`

# Extract all other packages as secondary ones, keeping only the binaries
tar -C /usr/local --wildcards -xzf `basename $DIST_LINUX_32` go/pkg/linux_386*
GOOS=linux GOARCH=386 /usr/local/go/pkg/tool/linux_amd64/dist bootstrap
tar -C /usr/local --wildcards -xzf `basename $DIST_LINUX_ARM` go/pkg/linux_arm*
GOOS=linux GOARCH=arm /usr/local/go/pkg/tool/linux_amd64/dist bootstrap

tar -C /usr/local --wildcards -xzf `basename $DIST_OSX_64` go/pkg/darwin_amd64*
GOOS=darwin GOARCH=amd64 /usr/local/go/pkg/tool/linux_amd64/dist bootstrap
tar -C /usr/local --wildcards -xzf `basename $DIST_OSX_32` go/pkg/darwin_386*
GOOS=darwin GOARCH=386 /usr/local/go/pkg/tool/linux_amd64/dist bootstrap

unzip -d /usr/local -q `basename $DIST_WIN_64` go/pkg/windows_amd64*
GOOS=windows GOARCH=amd64 /usr/local/go/pkg/tool/linux_amd64/dist bootstrap
unzip -d /usr/local -q `basename $DIST_WIN_32` go/pkg/windows_386*
GOOS=windows GOARCH=386 /usr/local/go/pkg/tool/linux_amd64/dist bootstrap

# Delete all the intermediate downloaded files
rm -f `basename $DIST_LINUX_64` `basename $DIST_LINUX_32` `basename $DIST_LINUX_ARM` \
      `basename $DIST_OSX_64` `basename $DIST_OSX_32`                                \
      `basename $DIST_WIN_64` `basename $DIST_WIN_32`
