#!/bin/sh

docker build -t tudorg/xgo-base base/ && \
    docker build -t tudorg/xgo-1.4.2 go-1.4.2/ &&
    docker build -t tudorg/beats-builder beats-builder
