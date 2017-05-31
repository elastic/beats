FROM tudorg/xgo-deb6-1.8.3

MAINTAINER Tudor Golubenco <tudor@elastic.co>

# Get libpcap-32 binaries from a DEB file
RUN \
	mkdir -p /libpcap && \
    wget http://archive.debian.org/debian/pool/main/libp/libpcap/libpcap0.8-dev_1.1.1-2+squeeze1_i386.deb && \
	dpkg -x libpcap0.8-dev_*_i386.deb /libpcap/i386 && \
	rm libpcap0.8-dev*.deb

# Get libpcap-64 binaries by compiling from source
RUN \
	apt-get -o Acquire::Check-Valid-Until=false update && \
	apt-get install -y flex bison
RUN ./fetch.sh http://www.tcpdump.org/release/libpcap-1.8.1.tar.gz 32d7526dde8f8a2f75baf40c01670602aeef7e39 && \
  mkdir -p /libpcap/amd64 && \
  tar -C /libpcap/amd64/ -xvf libpcap-1.8.1.tar.gz && \
  cd /libpcap/amd64/libpcap-1.8.1 && \
  ./configure --enable-usb=no --enable-bluetooth=no --enable-dbus=no && \
  make

# Old git version which does not support proxy with go get requires to fetch go-yaml directly
RUN git clone https://github.com/go-yaml/yaml.git /go/src/gopkg.in/yaml.v2

# Load gotpl which is needed for creating the templates.
RUN go get github.com/tsg/gotpl

# add patch for gopacket
ADD gopacket_pcap.patch /gopacket_pcap.patch
