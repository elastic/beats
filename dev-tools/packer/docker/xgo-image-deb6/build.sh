#!/bin/sh

docker build --rm=true -t tudorg/xgo-deb6-base base/ && \
    docker build --rm=true -t tudorg/xgo-deb6-1.9.4 go-1.9.4/ &&
    docker build --rm=true -t tudorg/beats-builder-deb6 beats-builder
