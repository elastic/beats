
BEATNAME?=packetbeat
BEAT_DESCRIPTION?=Packetbeat analyzes network traffic and sends the data to Elasticsearch.
SYSTEM_TESTS=true
TEST_ENVIRONMENT=false
TARGETS="windows/amd64 windows/386 darwin/amd64"
TARGETS_OLD="linux/amd64 linux/386"
CGO=true

include ../libbeat/scripts/Makefile

.PHONY: with_pfring
with_pfring:
	go build --tags havepfring

# This is called by the beats packer before building starts
.PHONY: before-build
before-build:
	sed -i.bk 's/device: any/device: en0/' $(PREFIX)/packetbeat-darwin.yml
	rm $(PREFIX)/packetbeat-darwin.yml.bk
	sed -i.bk 's/device: any/device: en0/' $(PREFIX)/packetbeat-darwin.full.yml
	rm $(PREFIX)/packetbeat-darwin.full.yml.bk
	# win
	sed -i.bk 's/device: any/device: 0/' $(PREFIX)/packetbeat-win.yml
	rm $(PREFIX)/packetbeat-win.yml.bk
	sed -i.bk 's/device: any/device: 0/' $(PREFIX)/packetbeat-win.full.yml
	rm $(PREFIX)/packetbeat-win.full.yml.bk

# Collects all dependencies and then calls update
.PHONY: collect
collect: fields

.PHONY: fields
fields:
	cat _meta/fields_base.yml > _meta/fields.yml
	cat protos/*/_meta/fields.yml >> _meta/fields.yml

.PHONY: benchmark
benchmark:
	go test -short -bench=. ./... -cpu=2
