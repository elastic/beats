#!/usr/bin/env bash

name=$(sha256sum go.mod | head -c 40)

if buildkite-agent artifact search "$name.tar.gz"; then
	echo "cache already up to date."
	exit 0
fi

echo "downloading deps"
go mod download

echo "running go mod tidy"
go mod tidy

ls -alh $(go env GOMODCACHE)

tar -czf "$name.tar.gz" -C $(go env GOMODCACHE) .

buildkite-agent artifact upload "$name.tar.gz"


