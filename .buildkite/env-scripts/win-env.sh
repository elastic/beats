#!/usr/bin/env bash

install_python_win() {
  if [[ ${PLATFORM_TYPE} = MINGW* ]]; then
    choco install mingw -y
    choco install python --version=3.11.0 -y
  fi
}
