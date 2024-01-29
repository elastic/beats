#!/usr/bin/env bash

install_python_win() {
  echo "PLATFORM TYPE: ${PLATFORM_TYPE}"
  if [[ ${PLATFORM_TYPE} = MINGW* ]]; then
    echo "Installing Python on Win"
    choco install mingw -y
    choco install python --version=3.11.0 -y
  fi
}

install_python_win
