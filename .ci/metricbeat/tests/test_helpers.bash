#!/usr/bin/env bats
# shellcheck shell=bash

# check dependencies
(
    type docker &>/dev/null || ( echo "docker is not available"; exit 1 )
    type curl &>/dev/null || ( echo "curl is not available"; exit 1 )
)>&2

function cleanup {
    docker kill "$1" &>/dev/null ||:
    docker rm -fv "$1" &>/dev/null ||:
}

function read_version {
    local file=$1
    local index=$2

    docker run --rm -v $(pwd):/workdir mikefarah/yq:2.4.0 yq r $file versions[$index]
}

function read_versions {
    local file=$1

    docker run --rm -v $(pwd):/workdir mikefarah/yq:2.4.0 yq r $file versions
}
