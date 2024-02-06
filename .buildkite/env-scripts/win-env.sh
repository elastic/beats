#!/usr/bin/env bash

echo "--- PLATFORM TYPE: ${PLATFORM_TYPE}"
if [[ ${PLATFORM_TYPE} = MINGW* ]]; then
  echo "--- Installing Python on Win"
  choco install mingw -y
  choco install python --version=3.11.0 -y
fi
