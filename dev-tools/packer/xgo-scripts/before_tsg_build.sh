#!/bin/sh

PB_PATH=/go/src/github.com/elastic/packetbeat
PB_TSG_PATH=/go/src/github.com/tsg/packetbeat

mkdir -p `dirname $PB_PATH`
ln -s $PB_TSG_PATH $PB_PATH

if [ -d $PB_PATH ]; then
	cd $PB_PATH
	patch -p1 < /gopacket_pcap.patch
fi
