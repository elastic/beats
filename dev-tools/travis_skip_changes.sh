#!/bin/bash
set -e

# commit range to check for. For example master...<PR branch>
RANGE=$1
shift
DIRLIST=$@

# find modified files in range and filter out docs only changes
CHANGED_FILES=$(git diff --name-only $RANGE | grep -v '.asciidoc')

beginswith() { case $2 in "$1"*) true;; *) false;; esac }

for path in $DIRLIST; do
  for changed in $CHANGED_FILES; do
    if beginswith $path $changed; then
      exit 1
    fi
  done
done
exit 0
