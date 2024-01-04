#!/usr/bin/env bash

set -euo pipefail

source .buildkite/filebeat/scripts/common.sh

echo ":: Evaluate Filebeat Changes ::"

changeset="^filebeat/
    ^go.mod
    ^pytest.ini
    ^dev-tools/
    ^libbeat/
    ^testing/
    ^\.buildkite/filebeat/"

if ! are_files_changed "$changeset" ; then
  message="No files changed within Filebeat changeset"
  echo "$message"
  buildkite-agent annotate "$message" --style 'warning' --context 'ctx-warn'
  # This should return any error but skip the release.
  exit 0
fi

# ToDo - remove after Beats agent is created"
echo ":: Setup Env ::"
add_bin_path
with_go
with_mage
# ToDo - end

echo ":: Start Packaging ::"
cd filebeat
umask 0022
mage package
