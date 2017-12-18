BEAT_NAME=heartbeat
BEAT_TITLE=Heartbeat
BEAT_PACKAGE_NAME=heartbeat-elastic
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
collect: fields imports

# Collects all module and metricset fields
.PHONY: fields
fields:
	@mkdir -p _meta/
	@cp ${ES_BEATS}/heartbeat/_meta/fields.common.yml _meta/fields.generated.yml
	@cat ${ES_BEATS}/heartbeat/monitors/active/*/_meta/fields.yml >> _meta/fields.generated.yml

# Generate imports for all monitors
.PHONY: imports
imports: python-env
	@mkdir -p include
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/script/generate_imports.py --out monitors/defaults/default.go ${BEAT_PATH}

# Create a new monitor. Requires the parameter MONITOR
.PHONY: create-monitor
create-monitor:
	go run scripts/generate_monitor/main.go -monitor=${MONITOR}
