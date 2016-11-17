FROM tudorg/xgo-1.7.1

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

# Add patch for gopacket.
ADD gopacket_pcap.patch /gopacket_pcap.patch

# Add the wpcap.dll from the WinPcap_4_1_2.exe installer so that
# we can generate a 64-bit compatible libwpcap.a.
ENV WINPCAP_DLL_SHA1 d2afb08d0379bd96e423857963791e2ba00c9645
ADD wpcap.dll /libpcap/win/wpcap.dll
RUN \
    apt-get install mingw-w64-tools && \
    cd /libpcap/win && \
    echo "$WINPCAP_DLL_SHA1 wpcap.dll" | sha1sum -c - && \
    gendef /libpcap/win/wpcap.dll && \
    x86_64-w64-mingw32-dlltool --as-flags=--64 -m i386:x86-64 -k --output-lib /libpcap/win/WpdPack/Lib/x64/libwpcap.a --input-def wpcap.def && \
    rm wpcap.def wpcap.dll

