#!/usr/bin/env bash

if [[ $PLATFORM_TYPE == Darwin* ]]; then
  echo ":: Setting larger ulimit on MacOS ::"
  # To bypass file descriptor errors like "Too many open files error" on MacOS
  ulimit -Sn 50000
  echo ":: ULIMIT :: $(ulimit -n)"
fi
