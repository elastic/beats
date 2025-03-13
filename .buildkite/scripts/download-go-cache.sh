#!/usr/bin/env bash

name=$(sha256sum go.mod | head -c 40)

buildkite-agent artifact download "$name.tar.gz" . --build "$BUILDKITE_TRIGGERED_FROM_BUILD_ID"

mkdir -p $(go env GOMODCACHE)

ls -alh $(go env GOMODCACHE)

tar -xf "$name.tar.gz" -C $(go env GOMODCACHE)

ls -alh $(go env GOMODCACHE)
