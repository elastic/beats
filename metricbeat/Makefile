#!/bin/bash

# Name can be overwritten, as Metricbeat is also a library
BEATNAME?=metricbeat
BEAT_DESCRIPTION?=Sends metrics to Elasticsearch.
SYSTEM_TESTS=true
TEST_ENVIRONMENT?=true
GOPACKAGES=$(shell go list ${BEAT_DIR}/... | grep -v /vendor/)
ES_BEATS?=..

TARGETS?="windows/amd64 windows/386 darwin/amd64"
TARGETS_OLD?="linux/amd64 linux/386"
CGO=true

# Metricbeat can only be cross-compiled on platforms not requiring CGO which
# are the same platforms where the system metrics (cpu, memory) are not
# implemented.
GOX_OS=solaris netbsd
GOX_FLAGS='-arch=amd64'


include ${ES_BEATS}/libbeat/scripts/Makefile

# Collects all module dashboards
.PHONY: kibana
kibana:
	-rm -r etc/kibana
	mkdir -p etc/kibana
	-cp -r module/*/_meta/kibana/ etc/kibana

# Collects all module and metricset fields
.PHONY: fields
fields:
	mkdir -p etc/
	cat ${ES_BEATS}/metricbeat/etc/_meta/fields_base.yml > etc/fields.yml
	. ${PYTHON_ENV}/bin/activate; python ${ES_BEATS}/metricbeat/scripts/fields_collector.py >> etc/fields.yml

# Collects all module docs
.PHONY: collect-docs
collect-docs: python-env
	-rm -r docs/modules
	mkdir -p docs/modules
	. ${PYTHON_ENV}/bin/activate; python ${ES_BEATS}/metricbeat/scripts/docs_collector.py

# Collects all module configs
.PHONY: configs
configs: python-env
	. ${PYTHON_ENV}/bin/activate; python ${ES_BEATS}/metricbeat/scripts/config_collector.py $(PWD) > etc/beat.yml
	. ${PYTHON_ENV}/bin/activate; python ${ES_BEATS}/metricbeat/scripts/config_collector.py --full $(PWD) > etc/beat.full.yml

# Generates imports for all modules and metricsets
.PHONY: imports
imports:
	mkdir -p include
	python ${ES_BEATS}/metricbeat/scripts/generate_imports.py > include/list.go

# This is called by the beats packer before building starts
.PHONY: before-build
before-build:

# Runs all collection steps and updates afterwards
.PHONY: collect
collect: fields collect-docs configs kibana imports update

# Creates a new metricset. Requires the params MODULE and METRICSET
.PHONY: create-metricset
create-metricset:
	python ${ES_BEATS}/metricbeat/scripts/create_metricset.py --path=$(PWD) --es_beats=$(ES_BEATS) --module=$(MODULE) --metricset=$(METRICSET)


# Generates the data.json example documents
.PHONY: generate-json
generate-json: build-image
	 ${DOCKER_COMPOSE} run beat go test -tags=integration github.com/elastic/beats/metricbeat/module/... -data
