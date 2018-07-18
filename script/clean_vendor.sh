#!/usr/bin/env bash

# Removes unnecessary files from the vendor directories
#
# A list for files to be removed is used instead of excluding files.
# The reason is that this makes the setup simpler and prevents
# from removing files by accident
#
# In general it should always be checked manually which files were removed.
# For example some projects like stretchr/testify have two LICENSE files
# with different names, where it is ok that one is removed.
#
# We keep the CHANGELOG in as this makes it easy visible when updating dependencies
# on what has changed.

# In general the following files should be kept:
# * .go
# * LICENSE* CHANGELOG*, PATENT*, CONTRIBUTORS*, README*
# * `.s`, `.c`, `.cpp`, `.cc`, `.c++`
# * `.m`, `.mm`, `.m++`

# Finds all vendor directories
DIR_LIST=`find . -type d -name vendor`

# Remove directories used for versioning
find $DIR_LIST -type d -name ".bzr" -o -name ".git" -exec rm -rf {} \;

## Removing all yaml files which are normally config files (travis.yml)
find $DIR_LIST -type f -name "*.yml" -exec rm -r {} \;

## Removing all golang test files
find $DIR_LIST -type f -name "*_test.go" -exec rm -r {} \;

## Removing all files starting with a dot (like .gitignore)
find $DIR_LIST -type f -name ".*" -exec rm -r {} \;

## Removing all .txt files which are normally test data or docs
## Excluding files mentioned above as e.x. nranchev/go-libGeoIP has the license in a .txt file
find $DIR_LIST -type f -name "*.txt" -a ! \( -iname "LICENSE.*" -o -iname "CHANGELOG.*" -o -iname "PATENT.*" -o -iname "CONTRIBUTORS.*" -o -iname "README.*" \) -exec rm -r {} \;

## Removing all *.cfg files
find $DIR_LIST -type f -name "*.cfg" -exec rm -r {} \;

## Removing all *.bat files
find $DIR_LIST -type f -name "*.bat" -exec rm -r {} \;

## Removing all *.tar files
find $DIR_LIST -type f -name "*.tar" -exec rm -r {} \;
