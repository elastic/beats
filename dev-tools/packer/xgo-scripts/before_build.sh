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

cd $BEAT_PATH

PREFIX=/build

# Add data to the home directory
mkdir -p $PREFIX/homedirs/$BEATNAME
make install-home HOME_PREFIX=$PREFIX/homedirs/$BEATNAME
if [ -n "BUILDID" ]; then
    echo "$BUILDID" > $PREFIX/homedirs/$BEATNAME/.build_hash.txt
fi

# Copy template
cp $BEATNAME.template.json $PREFIX/$BEATNAME.template.json
cp $BEATNAME.template-es2x.json $PREFIX/$BEATNAME.template-es2x.json

# linux
cp $BEATNAME.yml $PREFIX/$BEATNAME-linux.yml
cp $BEATNAME.full.yml $PREFIX/$BEATNAME-linux.full.yml

# darwin
cp $BEATNAME.yml $PREFIX/$BEATNAME-darwin.yml
cp $BEATNAME.full.yml $PREFIX/$BEATNAME-darwin.full.yml

# win
cp $BEATNAME.yml $PREFIX/$BEATNAME-win.yml
cp $BEATNAME.full.yml $PREFIX/$BEATNAME-win.full.yml

# Contains beat specific adjustments. As it is platform specific knowledge, it should be in packer not the beats itself
PREFIX=$PREFIX make before-build
