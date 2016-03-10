#!/bin/sh

set -e

BEATS_PATH=/go/src/${1}

# BEATNAME is in the $PACK variable
BEATNAME=$PACK

if [ $BEATNAME = "packetbeat" ]; then
	patch -p1 < /gopacket_pcap.patch
fi

if [ $BEATS_PATH = "/go/src/github.com/elastic/beats" ]; then
    BEAT_PATH=$BEATS_PATH/$BEATNAME
else
    BEAT_PATH=$BEATS_PATH
fi

echo $BEAT_PATH

cd $BEAT_PATH

PREFIX=/build

echo $PREFIX


# Copy template
cp $BEATNAME.template.json $PREFIX/$BEATNAME.template.json

# linux
cp $BEATNAME.yml $PREFIX/$BEATNAME-linux.yml

# Updates template path for linux distros
sed -i "s@path: \"$BEATNAME.template.json\"@path: \"/etc/$BEATNAME/$BEATNAME.template.json\"@" $PREFIX/$BEATNAME-linux.yml

# binary
cp $BEATNAME.yml $PREFIX/$BEATNAME-binary.yml

# darwin
cp $BEATNAME.yml $PREFIX/$BEATNAME-darwin.yml

# win
cp $BEATNAME.yml $PREFIX/$BEATNAME-win.yml

# Contains beat specific adjustments. As it is platform specific knowledge, it should be in packer not the beats itself
PREFIX=$PREFIX make before-build
