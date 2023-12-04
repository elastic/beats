#!/usr/bin/env bash

set -euo pipefail

are_files_changed() {
  changeset=$1

  if git diff --name-only HEAD@{1} HEAD | grep -qE "$changeset"; then
    return 0;
  else
    echo "WARN! No files changed in $changeset"
    return 1;
  fi
}
