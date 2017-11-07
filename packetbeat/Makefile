BEAT_NAME?=packetbeat
BEAT_TITLE?=Packetbeat
BEAT_DESCRIPTION?=Packetbeat analyzes network traffic and sends the data to Elasticsearch.
SYSTEM_TESTS?=true
TEST_ENVIRONMENT=false
ES_BEATS?=..

include ${ES_BEATS}/libbeat/scripts/Makefile

# This is called by the beats packer before building starts
.PHONY: before-build
before-build:
	sed -i.bk 's/device: any/device: en0/' $(PREFIX)/packetbeat-darwin.yml
	rm $(PREFIX)/packetbeat-darwin.yml.bk
	sed -i.bk 's/device: any/device: en0/' $(PREFIX)/packetbeat-darwin.reference.yml
	rm $(PREFIX)/packetbeat-darwin.reference.yml.bk
	# win
	sed -i.bk 's/device: any/device: 0/' $(PREFIX)/packetbeat-win.yml
	rm $(PREFIX)/packetbeat-win.yml.bk
	sed -i.bk 's/device: any/device: 0/' $(PREFIX)/packetbeat-win.reference.yml
	rm $(PREFIX)/packetbeat-win.reference.yml.bk

# Collects all dependencies and then calls update
.PHONY: collect
collect: fields imports

.PHONY: fields
fields:
	@mkdir -p _meta
	@cp ${ES_BEATS}/packetbeat/_meta/fields_base.yml _meta/fields.yml
	@-cat protos/*/_meta/fields.yml >> _meta/fields.yml

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
