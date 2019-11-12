#!/bin/sh

function push_module() {
  local module=$1

  docker push "docker.elastic.co/observability-ci/${module}"
}

function main() {
  push_module "${@}"
}

main "${@}"