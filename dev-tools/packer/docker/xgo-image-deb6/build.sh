#!/bin/sh

docker build --rm=true -t tudorg/xgo-deb6-base base/ && \
    docker build --rm=true -t tudorg/xgo-deb6-1.7.1 go-1.7.1/ &&
    docker build --rm=true -t tudorg/beats-builder-deb6 beats-builder
