#!/bin/sh

docker pull tudorg/xgo-base:v20180222 && \
    docker build --rm=true -t tudorg/xgo-1.9.2 go-1.9.2/ &&
    docker build --rm=true -t tudorg/beats-builder beats-builder
