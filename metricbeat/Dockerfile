FROM golang:1.6
MAINTAINER Nicolas Ruflin <ruflin@elastic.co>

RUN set -x && \
    apt-get update && \
    apt-get install -y netcat python-virtualenv python-pip && \
    apt-get clean


## Install go package dependencies
RUN set -x \
  go get \
	github.com/pierrre/gotestcover \
	github.com/tsg/goautotest \
	golang.org/x/tools/cmd/vet

# Setup work environment
ENV METRICBEAT_PATH /go/src/github.com/elastic/beats/metricbeat
ENV GO15VENDOREXPERIMENT=1

RUN mkdir -p $METRICBEAT_PATH/build/coverage
WORKDIR $METRICBEAT_PATH
