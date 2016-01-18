FROM golang:1.5.2
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

ENV GO15VENDOREXPERIMENT=1

RUN mkdir -p /etc/pki/tls/certs
COPY testing/environments/docker/logstash/pki/tls/certs/logstash.crt /etc/pki/tls/certs/logstash.crt

# Create a copy of the respository inside the container.
COPY . /go/src/github.com/elastic/beats/
