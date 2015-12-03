#!/usr/bin/env bash

set -e

# Takes first entry in GOPATH in case has multiple entries
GOPATH=`echo $GOPATH | tr ':' '\n' | head -1`
BEATNAME=$1
BEATPATH=$2
LIBBEAT=${GOPATH}/src/github.com/elastic/beats/libbeat

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
