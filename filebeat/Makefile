#!/bin/bash

BEATNAME?=filebeat
BEAT_DESCRIPTION?=Filebeat sends log files to Logstash or directly to Elasticsearch.
SYSTEM_TESTS=true
TEST_ENVIRONMENT=false
GOX_FLAGS='-arch=amd64 386 arm ppc64 ppc64le'

include ../libbeat/scripts/Makefile

# This is called by the beats packer before building starts
.PHONY: before-build
before-build:

# Collects all dependencies and then calls update
.PHONY: collect
collect:
