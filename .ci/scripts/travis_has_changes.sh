#!/usr/bin/env bash
set -exuo pipefail

# Changes on these files will trigger all builds.
COMMON_DIRLIST="dev-tools .travis.yml testing .ci"

# Commit range to check for. For example master...<PR branch>
RANGE=$TRAVIS_COMMIT_RANGE
DIRLIST="$@ $COMMON_DIRLIST"

# Find modified files in range and filter out docs only changes.
CHANGED_FILES=$(git diff --name-only $RANGE | grep -v '.asciidoc')

beginswith() { case $2 in "$1"*) true;; *) false;; esac }

for path in $DIRLIST; do
  for changed in $CHANGED_FILES; do
    if beginswith $path $changed; then
      exit 0 # found a match -> continue testing
    fi
  done
done

echo "NOT testing required. Modified files: $CHANGED_FILES"
exit 1
