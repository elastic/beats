#!/bin/bash
#
# Contains the a dependency builder to iterate over all installed dependencies
# and cross compile them to the requested target platform.
# 
# Usage: build_deps.sh <dependency folder>
#
# Needed environment variables:
#   CC     - C cross compiler to use for the build
#   HOST   - Target platform to build (used to find the needed tool-chains)
#   PREFIX - File-system path where to install the built binaries
set -e

# Remove any previous build leftovers, and copy a fresh working set (clean doesn't work for cross compiling)
rm -rf /deps-build && cp -r $1 /deps-build

# Build all the dependencies (no order for now)
for dep in `ls /deps-build`; do
	echo "Configuring dependency $dep for $HOST..."
	(cd /deps-build/$dep && ./configure --disable-shared --host=$HOST --prefix=$PREFIX --silent)

	echo "Building dependency $dep for $HOST..."
	(cd /deps-build/$dep && make --silent -j install)
done

# Remove any build artifacts
rm -rf /deps-build
