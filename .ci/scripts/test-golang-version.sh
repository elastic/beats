#!/usr/bin/env bash
if grep -q "$1" .go-version ; then
  echo "found $1. No need to do nothing"
  # return false
  exit 1
fi
echo "not found $1. Need to update the file"
# return true
exit 0
