#!/bin/bash

BEATNAME?=filebeat
BEAT_DESCRIPTION?=Sends log files to Logstash or directly to Elasticsearch.
SYSTEM_TESTS=true
TEST_ENVIRONMENT=false

include ../libbeat/scripts/Makefile

# This is called by the beats packer before building starts
.PHONY: before-build
before-build:
