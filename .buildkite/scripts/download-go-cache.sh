#!/usr/bin/env bash

name=$(sha256sum go.mod | head -c 40)

buildkite-agent artifact download "$name.tar.gz" . --build "$BUILDKITE_TRIGGERED_FROM_BUILD_ID"

folder="$(go env GOMODCACHE)/cache"

mkdir -p "$folder"

id

ls -alh $(go env GOMODCACHE)

echo "extracting cache archive to $folder"
tar -xf "$name.tar.gz" --skip-old-files -C "$folder"

ls -alh $(go env GOMODCACHE)
