#!/bin/bash

###############################################################################
# Cross-compilation utility script for Go. This utility does not support cross-
# compiling applications that use cgo. Go 1.5 supports cross compiling
# without having to bootstrap the Go tool chain so all you need to do is source
# this file in bash and call go-build-all.
#
# Examples:
#
# Build for all platforms and put the outputs into the current dir.
#   source crosscompile.bash
#   go-build-all
#
# Build for all platforms and put the outputs into the ./bin dir.
#   source crosscompile.bash
#   OUT=./bin go-build-all
#
# Build for a single platform.
#   source crosscompile.bash
#   go-solaris-amd64 build -o output-filename
###############################################################################

# List of all platforms types. Comment out (with #) any platforms that you want
# nacl and plan9 are currently not compatible
# If you want to build only a specific platform, set the PLATFORMS variable
# go-build-all to ignore.
: ${PLATFORMS:="
linux-386
linux-386-387
linux-arm-arm5
linux-amd64
linux-arm
linux-arm64
linux-ppc64
linux-ppc64le
#nacl-386
#nacl-amd64p32
#nacl-arm
darwin-386
darwin-amd64
dragonfly-amd64
freebsd-386
freebsd-amd64
freebsd-arm
netbsd-386
netbsd-amd64
netbsd-arm
openbsd-386
openbsd-amd64
openbsd-arm
#plan9-386
#plan9-amd64
solaris-amd64
windows-386
windows-amd64
"}

# Build for all platforms. Any failures will be reported at the end.
function go-build-all {
    local DIR=${OUT:-.}
    mkdir -p "$DIR"
    local FAILURES=""
    for PLATFORM in $(echo "$PLATFORMS" | grep -v ^#); do
        local SRCFILENAME=`echo $@ | sed 's/\.go//'`
        local CURDIRNAME=${PWD##*/}
        local OUTPUT=${SRCFILENAME:-$CURDIRNAME} # if no src file given, use current dir name
        if [[ "$PLATFORM" =~ "windows" ]]; then local EXT=".exe"; fi
        local CMD="go-${PLATFORM} build -o ${DIR}/$OUTPUT-${PLATFORM}${EXT} $@"
        echo $CMD
        $CMD || FAILURES="$FAILURES $PLATFORM"
    done
    if [ "$FAILURES" != "" ]; then
        echo "*** go-build-all FAILED on$FAILURES ***"
        return 1
    fi
}

# Define a function that calls 'godep go' with the GOOS and GOARCH environment
# variables assiciated with the given platform.
function go-alias {
    local GOOS=$(echo $1 | sed 's/-.*//')
    local GOARCH=$(echo $1 | sed 's/.*-//')
    local CMD="GOOS=${GOOS} GOARCH=${GOARCH}"
    if [ "$GOARCH" = "arm5" ]; then
        local GOARCH=arm
        local GOARM=5
        CMD="GOOS=${GOOS} GOARCH=${GOARCH} GOARM=${GOARM}"
    fi
    if [ "$GOARCH" = "387" ]; then
        local GOARCH=386
        local GO386=387
        CMD="GOOS=${GOOS} GOARCH=${GOARCH} GO386=${GO386}"
    fi
    CMD="$CMD godep go"
    eval "function go-${1} { ( ${CMD} \"\$@\" ) }"
}

# Create aliases for each platform type.
for PLATFORM in $PLATFORMS; do
    go-alias ${PLATFORM#\#}
done

# Removed the go-alias function.
unset -f go-alias
