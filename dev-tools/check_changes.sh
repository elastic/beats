#!/usr/bin/env bash
set -e

# commit range to check for. For example master...<PR branch>
RANGE=$1
shift
DIRLIST=$@
CHANGED_FILES=$(git diff --name-only $RANGE)

beginswith() { case $2 in "$1"*) true;; *) false;; esac }

for path in $DIRLIST; do
  for changed in $CHANGED_FILES; do
    if beginswith $path $changed; then
      exit 0
    fi
  done
done
exit 1