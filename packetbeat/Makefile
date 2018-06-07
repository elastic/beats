BEAT_NAME?=packetbeat
BEAT_TITLE?=Packetbeat
SYSTEM_TESTS?=true
TEST_ENVIRONMENT=false
ES_BEATS?=..
FIELDS_FILE_PATH=protos

include ${ES_BEATS}/libbeat/scripts/Makefile

# Collects all dependencies and then calls update
.PHONY: collect
collect: fields imports

.PHONY: benchmark
benchmark:
	go test -short -bench=. ./... -cpu=2

.PHONY: create-tcp-protocol
create-tcp-protocol: python-env
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/packetbeat/scripts/create_tcp_protocol.py

# Generates imports for all modules and metricsets
.PHONY: imports
imports: python-env
	@mkdir -p include
	@mkdir -p protos
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/script/generate_imports.py ${BEAT_PATH}
