#!/bin/sh
cp -r ../../../vendor/gopkg.in/yaml.v2 beats-builder/yaml.v2
cp -r ../../../vendor/github.com/tsg/gotpl beats-builder/gotpl
docker build --rm=true -t tudorg/xgo-deb6-base base/ && \
    docker build --rm=true -t tudorg/xgo-deb6-1.9.4 go-1.9.4/ &&
    docker build --rm=true -t tudorg/beats-builder-deb6 beats-builder
