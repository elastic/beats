#!/usr/bin/env bash

name=$(sha256sum go.mod | head -c 40)

buildkite-agent artifact download "$name.tar.gz" . --build "$BUILDKITE_TRIGGERED_FROM_BUILD_ID"

tar -xf "$name.tar.gz" -C $(go env GOMODCACHE)
