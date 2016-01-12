#!/bin/sh

PB_PATH=/go/src/github.com/elastic/beats/

if [ -d $PB_PATH ]; then
	cd $PB_PATH
	patch -p1 < /gopacket_pcap.patch
    cd packetbeat/
    PREFIX=/build make install-cfg
fi
