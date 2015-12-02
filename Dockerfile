FROM golang:1.5.1
MAINTAINER Nicolas Ruflin <ruflin@elastic.co>

RUN set -x && \
    apt-get update && \
    apt-get install -y netcat && \
    apt-get clean


## Install go package dependencies
RUN set -x \
  go get \
	github.com/pierrre/gotestcover \
	github.com/tsg/goautotest \
	golang.org/x/tools/cmd/cover \
	golang.org/x/tools/cmd/vet

COPY libbeat/scripts/docker-entrypoint.sh /entrypoint.sh

# Setup work environment
ENV LIBBEAT_PATH /go/src/github.com/elastic/beats/libbeat
ENV GO15VENDOREXPERIMENT=1
RUN mkdir -p $LIBBEAT_PATH/coverage
WORKDIR $LIBBEAT_PATH

RUN mkdir -p /etc/pki/tls/certs
COPY libbeat/scripts/docker/logstash/pki/tls/certs/logstash.crt /etc/pki/tls/certs/logstash.crt

# Create a copy of the respository inside the container.
COPY . /go/src/github.com/elastic/beats/

# It is expected that libbeat from the host is mounted
# within the container at the WORKDIR location.
ENTRYPOINT ["/entrypoint.sh"]

# Build libbeat inside of the container so that it is ready
# for testing.
RUN make
