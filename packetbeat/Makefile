
BEATNAME?=packetbeat
BEAT_DESCRIPTION?=A network analyzer tool that sends data to Elasticsearch.
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

.PHONY: benchmark
benchmark:
	go test -short -bench=. ./... -cpu=2
