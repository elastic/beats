FROM golang:1.5.1
MAINTAINER Nicolas Ruflin <spam@ruflin.com>

# Install required packages
RUN apt-get update && apt-get install -y \
	geoip-database \
	libpcap-dev \
	python-pip \
	python-virtualenv

# Install go package dependencies
RUN go get \
	github.com/tools/godep \
	github.com/pierrre/gotestcover \
	golang.org/x/tools/cmd/cover \
	golang.org/x/tools/cmd/vet

# Setup work environment
RUN mkdir -p /go/src/github.com/elastic/packetbeat
WORKDIR /go/src/github.com/elastic/packetbeat

COPY . /go/src/github.com/elastic/packetbeat

# Make sure to clean up environment first
RUN make clean
RUN make -C tests/ clean

# Build base environment
RUN make
RUN make packetbeat.test
