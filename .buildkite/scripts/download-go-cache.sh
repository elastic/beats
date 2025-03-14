#!/usr/bin/env bash

name=$(sha256sum go.mod | head -c 40)

buildkite-agent artifact download "$name.tar.gz" . --build "$BUILDKITE_TRIGGERED_FROM_BUILD_ID"

mkdir -p $(go env GOMODCACHE)

id

ls -alh $(go env GOMODCACHE)

echo "extracting cache archive to $(go env GOMODCACHE)/cache"
tar -xf "$name.tar.gz" --skip-old-files -C "$(go env GOMODCACHE)/cache"

ls -alh $(go env GOMODCACHE)
