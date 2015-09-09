#!/bin/bash
#
# Contains a simple fetcher to download a file from a remote URL and verify its
# SHA1 checksum.
#
# Usage: fetch.sh <remote URL> <SHA1 checksum>
set -e

# Pull the file from the remote URL
file=`basename $1`
echo "Downloading $1..."
wget --no-check-certificate -q $1

# Generate a desired checksum report and check against it
echo "$2  $file" > $file.sum
sha1sum -c $file.sum
rm $file.sum
