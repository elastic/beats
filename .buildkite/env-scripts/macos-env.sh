#!/usr/bin/env bash

if [[ "$BUILDKITE_STEP_KEY" == macos* ]]; then
  echo ":: Setting larger ulimit on MacOS ::"
  # To bypass file descriptor errors like "Too many open files error" on MacOS
  ulimit -Sn 10000
fi
