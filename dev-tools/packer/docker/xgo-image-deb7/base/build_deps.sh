#!/bin/bash
#
# Contains the dependency builder to iterate over all installed dependencies
# and cross compile them to the requested target platform.
# 
# Usage: build_deps.sh dependency_root_folder dependency1 dependency_2 ...
#
# Needed environment variables:
#   CC     - C cross compiler to use for the build
#   HOST   - Target platform to build (used to find the needed tool-chains)
#   PREFIX - File-system path where to install the built binaries
#   STATIC - true if the libraries are statically linked to the go application
set -e

DEP_ROOT_FOLDER=$1

# Remove any previous build leftovers, and copy a fresh working set (clean doesn't work for cross compiling)
rm -rf /deps-build && cp -r $DEP_ROOT_FOLDER /deps-build

args=("$@")

if [ "$STATIC" == "true" ]; then DISABLE_SHARED=-disable-shared; fi

# Build all the dependencies
for ((i=1; i<${#args[@]}; i++)); do
	dep=${args[i]}
	echo "Configuring dependency $dep for $HOST..."
	if [ -f "/deps-build/$dep/autogen.sh" ]; then (cd /deps-build/$dep && ./autogen.sh); fi
	(cd /deps-build/$dep && ./configure $DISABLE_SHARED --host=$HOST --prefix=$PREFIX --silent)

	echo "Building dependency $dep for $HOST..."
	(cd /deps-build/$dep && make --silent -j install)
done

# Remove any build artifacts
rm -rf /deps-build
