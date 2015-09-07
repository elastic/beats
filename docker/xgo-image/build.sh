#!/bin/sh

docker build --rm=true -t tudorg/xgo-base base/ && \
    docker build --rm=true -t tudorg/xgo-1.4.2 go-1.4.2/ &&
    docker build --rm=true -t tudorg/beats-builder beats-builder
