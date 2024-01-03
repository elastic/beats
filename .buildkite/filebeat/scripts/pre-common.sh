#!/usr/bin/env bash

prepare_win() {
  local os
  os="$(uname)"
  if [[ $os = MINGW* ]]; then
    choco install mingw -y
    choco install python --version=3.11.0 -y
  fi
}

check_filebeat_changes() {
  changeset=$1

  if git diff --name-only HEAD@{1} HEAD | grep -qE "$changeset"; then
    export FILEBEAT_CHANGESET=true
  else
    export FILEBEAT_CHANGESET=false
  fi
}
