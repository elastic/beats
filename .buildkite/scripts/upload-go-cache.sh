#!/usr/bin/env bash

name=$(sha256sum go.mod | head -c 40)

if buildkite-agent artifact search "$name.tar.gz"; then
	echo "cache already up to date."
	exit 0
fi

go mod download

tar -czvf "$name.tar.gz" $(go env GOMODCACHE)

buildkite-agent artifact upload "$name.tar.gz"


