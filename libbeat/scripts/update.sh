#!/usr/bin/env bash

set -e

BEATNAME=$1

# Setup
if [ -z $BEATNAME ]; then
    echo "beat name must be set"
    exit;
fi

BEATPATH=../$BEATNAME

if [ ! -d "$BEATPATH" ]; then
  echo "Beat dir does not exist: $BEATPATH"
  exit;
fi

LIBPATH=../libbeat

echo "Beat name: $BEATNAME"
echo "Beat path: $BEATPATH"
echo "libbeat path: $LIBPATH"

cd $BEATPATH


echo "Start modifying beat"

# Update config
echo "Update config file"
rm -f etc/$BEATNAME.yml
cat etc/beat.yml ${LIBPATH}/etc/libbeat.yml | sed -e "s/beatname/$BEATNAME/g" > etc/$BEATNAME.yml
