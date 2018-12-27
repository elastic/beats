#!/usr/bin/env bash
set -exuo pipefail

source ./dev-tools/common.bash

jenkins_setup

mage GoUnitTest || echo -e "\033[31;49mTests FAILED\033[0m"
