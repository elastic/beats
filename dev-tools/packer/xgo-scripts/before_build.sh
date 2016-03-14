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

echo $PREFIX

cp etc/$BEATNAME.template.json $PREFIX/$BEATNAME.template.json
# linux
cp $BEATNAME.yml $PREFIX/$BEATNAME-linux.yml
# binary
cp $BEATNAME.yml $PREFIX/$BEATNAME-binary.yml
# darwin
cp $BEATNAME.yml $PREFIX/$BEATNAME-darwin.yml
# win
cp $BEATNAME.yml $PREFIX/$BEATNAME-win.yml


# Contains beat specific adjustments. As it is platform specific knowledge, it should be in packer not the beats itself
PREFIX=$PREFIX make before-build
