#!/usr/bin/env bash

set -exuo pipefail

${HOME}/bin/docker-compose -f .ci/jobs/docker-compose.yml up -d

exit $?
