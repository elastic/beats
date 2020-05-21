#!/usr/bin/env bash
set -exuo pipefail

FOLDER=${1:-.build}

if [ -d "${FOLDER}" ] ; then
    rm -rf "${FOLDER}"
fi
mkdir -p "${FOLDER}"
find . -name "build" -type d -print0 | while read -r  -d $'\0' build
do
    base=$(basename "$(dirname "${build}")")
    mkdir -p "${FOLDER}/${base}"
    cp -rf "${build}" "${FOLDER}/${base}"
done