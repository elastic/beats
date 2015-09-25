FROM tudorg/xgo-1.5.1

MAINTAINER Tudor Golubenco <tudor@elastic.co>

# Get libpcap binaries for linux
RUN \
	dpkg --add-architecture i386 && \
	apt-get update && \
	apt-get install -y libpcap0.8-dev

RUN \
	mkdir -p /libpcap && \
	apt-get download libpcap0.8-dev:i386 && \
	dpkg -x libpcap0.8-dev_*_i386.deb /libpcap/i386 && \
	apt-get download libpcap0.8-dev && \
	dpkg -x libpcap0.8-dev_*_amd64.deb /libpcap/amd64 && \
	rm libpcap0.8-dev*.deb


# Get libpcap binaries for win
ENV WPDPACK_URL https://www.winpcap.org/install/bin/WpdPack_4_1_2.zip
RUN \
	./fetch.sh $WPDPACK_URL f5c80885bd48f07f41833d0f65bf85da1ef1727a && \
	unzip `basename $WPDPACK_URL` -d /libpcap/win && \
	rm `basename $WPDPACK_URL`

# add patch for gopacket
ADD gopacket_pcap.patch /gopacket_pcap.patch
