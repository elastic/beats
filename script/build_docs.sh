#!/usr/bin/env bash

set -e

name=$1
path=$2
build_dir=$3

docs_dir=$build_dir/docs
html_dir=$build_dir/html_docs

# Checks if docs clone already exists
if [ ! -d $docs_dir ]; then
    # Only head is cloned
    git clone --depth=1 https://github.com/elastic/docs.git $docs_dir
else
    echo "$docs_dir already exists. Not cloning."
fi

index_list="$(find ${GOPATH%%:*}/src/$path -name 'index.asciidoc' -maxdepth 1)"
for index in $index_list
do
  echo "Building docs for ${name}..."
  echo "Index document: ${index}"
  index_path=$(basename $(dirname $index))
  echo "Index path: $index_path"

  dest_dir="$html_dir/${name}/${index_path}"
  mkdir -p "$dest_dir"
  params="--chunk=1"
  if [ "$PREVIEW" = "1" ]; then
    params="--chunk=1 -open chunk=1 -open"
  fi
  $docs_dir/build_docs.pl $params --doc "$index" -out "$dest_dir"
done
