#!/bin/sh

docker build -t tudorg/xgo-deb6-base base/ && \
    docker build -t tudorg/xgo-deb6-1.4.2 go-1.4.2/ &&
    docker build -t tudorg/beats-builder-deb6 beats-builder
