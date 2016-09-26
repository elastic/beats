#!/bin/sh

docker build --rm=true -t tudorg/xgo-base base/ && \
    docker build --rm=true -t tudorg/xgo-1.7.1 go-1.7.1/ &&
    docker build --rm=true -t tudorg/beats-builder beats-builder
