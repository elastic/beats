#!/usr/bin/env bash

set -e

BEATNAME=$1
BEATPATH=$2
LIBBEAT=$3

# Setup
if [ -z $BEATNAME ]; then
    echo "beat name must be set"
    exit;
fi

if [ -z $BEATPATH ]; then
    echo "beat path must be set"
    exit;
fi

DIR=../$BEATNAME

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
rm -f etc/$BEATNAME.yml
cat etc/beat.yml ${LIBBEAT}/etc/libbeat.yml | sed -e "s/beatname/$BEATNAME/g" > $BEATNAME.yml
