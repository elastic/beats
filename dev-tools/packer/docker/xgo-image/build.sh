#!/bin/sh

docker build --rm=true -t tudorg/xgo-base base/ && \
    docker build --rm=true -t tudorg/xgo-1.6 go-1.6/ &&
    docker build --rm=true -t tudorg/beats-builder beats-builder
