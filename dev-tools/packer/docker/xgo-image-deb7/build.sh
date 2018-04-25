#!/bin/sh
cp -r ../../../vendor/gopkg.in/yaml.v2 beats-builder/yaml.v2
cp -r ../../../vendor/github.com/tsg/gotpl beats-builder/gotpl
docker build --rm=true -t tudorg/xgo-deb7-base base/ && \
    docker build --rm=true -t tudorg/xgo-deb7-1.10.1 go-1.10.1/ &&
    docker build --rm=true -t tudorg/beats-builder-deb7 beats-builder
