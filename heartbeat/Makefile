BEAT_NAME=heartbeat
BEAT_DESCRIPTION?=Ping remote services for availability and log results to Elasticsearch or send to Logstash.
SYSTEM_TESTS=true
TEST_ENVIRONMENT=false

# Path to the libbeat Makefile
-include ../libbeat/scripts/Makefile

# This is called by the beats packer before building starts
.PHONY: before-build
before-build:

# Collects all dependencies and then calls update
.PHONY: collect
collect: fields


# Collects all module and metricset fields
.PHONY: fields
fields: python-env
	mkdir -p _meta/
	cat ${ES_BEATS}/heartbeat/_meta/fields.common.yml > _meta/fields.generated.yml
	cat ${ES_BEATS}/heartbeat/monitors/active/*/_meta/fields.yml >> _meta/fields.generated.yml
