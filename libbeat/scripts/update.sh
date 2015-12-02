#!/usr/bin/env bash

set -e

# Takes first entry in GOPATH in case has multiple entries
GOPATH=`echo $GOPATH | tr ':' '\n' | head -1`
BEATNAME=$1
BEATPATH=$2
LIBBEAT=${GOPATH}/src/github.com/elastic/libbeat

# Setup
if [ -z $BEATNAME ]; then
    echo "beat name must be set"
    exit;
fi

if [ -z $BEATPATH ]; then
    echo "beat path must be set"
    exit;
fi

DIR=$GOPATH/src/$BEATPATH

if [ ! -d "$DIR" ]; then
  echo "Beat dir does not exist: $DIR"
  exit;
fi

echo "Beat name: $BEATNAME"
echo "Beat path: $DIR"

cd $DIR


echo "Start modifying beat"

# Update config
echo "Update config file"
rm etc/$BEATNAME.yml
cat etc/beat.yml ${LIBBEAT}/etc/libbeat.yml > etc/$BEATNAME.yml
sed -i "" -e s/beatname/$BEATNAME/g etc/$BEATNAME.yml

# Update Libbeat
echo "Update libbeat deps"
godep update github.com/elastic/libbeat/...

echo "Update .editorconfig"
cat ${LIBBEAT}/.editorconfig > .editorconfig

echo "Update .gitattributes"
cat ${LIBBEAT}/.gitattributes > .gitattributes

echo "Update LICENSE"
cat ${LIBBEAT}/LICENSE > LICENSE

echo "Update MAKEFILE"
cat ${LIBBEAT}/scripts/Makefile > scripts/Makefile
sed -i "" -e s/libbeat.test/$BEATNAME.test/g scripts/Makefile
sed -i "" -e "s/.PHONY: build$/.PHONY: $BEATNAME/" scripts/Makefile
sed -i "" -e "s/^build:/$BEATNAME:/" scripts/Makefile

echo "Update crosscompile.bash"
cat ${LIBBEAT}/scripts/crosscompile.bash > scripts/crosscompile.bash
