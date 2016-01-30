# Beats dockerfile used for testing
FROM golang:1.5.3
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

ENV GO15VENDOREXPERIMENT=1

RUN mkdir -p /etc/pki/tls/certs
