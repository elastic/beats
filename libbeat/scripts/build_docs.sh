#!/usr/bin/env bash

# Checks if docs clone already exists
if [ ! -d "build/docs" ]; then
    # Only head is cloned
    git clone --depth=1 https://github.com/elastic/docs.git build/docs
else
    echo "build/docs already exists. Not cloning."
fi

# beatnames must be passed as parameters. Example: packetbeat filebeat
for name in $*
do
  index="$GOPATH/src/github.com/elastic/beats/${name}/docs/index.asciidoc"
  echo $index
  if [ -f "$index" ]; then
    echo "Building docs for ${name}..."
    dest_dir="build/html_docs/${name}"
    mkdir -p "$dest_dir"
    ./build/docs/build_docs.pl --doc "$index" -out "$dest_dir"
  fi
done
