#!/bin/sh

docker build --rm=true -t tudorg/xgo-base base/ && \
    docker build --rm=true -t tudorg/xgo-1.8.1 go-1.8.1/ &&
    docker build --rm=true -t tudorg/beats-builder beats-builder
