FROM tudorg/xgo-deb6-1.7

MAINTAINER Tudor Golubenco <tudor@elastic.co>

# Get libpcap binaries for linux
RUN \
	mkdir -p /libpcap && \
    wget http://archive.debian.org/debian/pool/main/libp/libpcap/libpcap0.8-dev_1.1.1-2+squeeze1_i386.deb && \
	dpkg -x libpcap0.8-dev_*_i386.deb /libpcap/i386 && \
	wget http://archive.debian.org/debian/pool/main/libp/libpcap/libpcap0.8-dev_1.1.1-2+squeeze1_amd64.deb && \
	dpkg -x libpcap0.8-dev_*_amd64.deb /libpcap/amd64 && \
	rm libpcap0.8-dev*.deb
RUN \
	apt-get -o Acquire::Check-Valid-Until=false update && \
	apt-get install -y libpcap0.8-dev

# add patch for gopacket
ADD gopacket_pcap.patch /gopacket_pcap.patch
