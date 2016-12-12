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

# Collects all module and dataset fields
.PHONY: fields
fields:
	mkdir -p _meta/
	cat ${ES_BEATS}/filebeat/_meta/fields.common.yml > _meta/fields.generated.yml
	. ${PYTHON_ENV}/bin/activate; python ${ES_BEATS}/metricbeat/scripts/fields_collector.py >> _meta/fields.generated.yml


# Runs all collection steps and updates afterwards
.PHONY: collect
collect: fields update
