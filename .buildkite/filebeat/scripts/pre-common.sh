#!/usr/bin/env bash

prepare_win() {
  local os
  os="$(uname)"
  if [[ $os = MINGW* ]]; then
    choco install mingw -y
    choco install python --version=3.11.0 -y
  fi
}
