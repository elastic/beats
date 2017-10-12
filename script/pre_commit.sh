#!/usr/bin/env bash

set -e

echo "-- pre commit hook running"
#return if no files staged for commit
staged_files=$(git diff --name-only --cached)
[ -z "$staged_files" ] && exit 0

#run make cmds and check whether 
#- unstaged files have changed
#- make cmd fails
unstaged_files=$(git diff --name-only)

echo "---- lint"
make lint

echo "---- format"
make fmt

echo "---- misspell"
make misspell

unstaged_files_after=$(git diff --name-only)
if [ "$unstaged_files" == "$unstaged_files_after"  ] ; then
  exit 0
fi;
echo "Pre-Commit hook has failed, see changed files."
exit 1
