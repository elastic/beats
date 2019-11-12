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
    local module=$1
    local file=$2
    local index=$3

    docker run --rm -v "${EXCHANGE_PATH}/metricbeat/module/${module}/_meta:/workdir" mikefarah/yq:2.4.0 yq r $file versions[$index]
}

function read_versions {
    local module=$1
    local file=$2

    docker run --rm -v "${EXCHANGE_PATH}/metricbeat/module/${module}/_meta:/workdir" mikefarah/yq:2.4.0 yq r $file versions
}
