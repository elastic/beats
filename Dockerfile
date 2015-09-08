FROM golang:1.4
MAINTAINER Nicolas Ruflin <ruflin@elastic.co>

## Install go package dependencies
RUN go get \
	github.com/pierrre/gotestcover \
	github.com/tools/godep \
	github.com/tsg/goautotest \
	golang.org/x/tools/cmd/cover \
	golang.org/x/tools/cmd/vet

WORKDIR /go/src/github.com/elastic/libbeat
# Setup work environment
RUN mkdir -p /go/src/github.com/elastic/libbeat

COPY . /go/src/github.com/elastic/libbeat

RUN make
