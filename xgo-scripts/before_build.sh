#!/bin/sh

PB_PATH=/go/src/github.com/elastic/packetbeat

if [ -d $PB_PATH ]; then
	cd $PB_PATH
	patch -p1 < /gopacket_pcap.patch
fi
