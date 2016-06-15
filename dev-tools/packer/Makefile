BUILDID=$(shell git rev-parse HEAD)

.PHONY: all
all: packetbeat/deb packetbeat/rpm packetbeat/darwin packetbeat/win packetbeat/bin \
	filebeat/deb filebeat/rpm filebeat/darwin filebeat/win filebeat/bin \
	metricbeat/deb metricbeat/rpm metricbeat/darwin metricbeat/win metricbeat/bin \
	winlogbeat/win \
	build/upload/build_id.txt

.PHONY: packetbeat metricbeat
packetbeat metricbeat: build/upload
	@echo Cross-compiling $@
	# cross compile on ubuntu
	docker run --rm \
		-v $(abspath build):/build \
		-v $(abspath xgo-scripts):/scripts \
		-v $(abspath ../..):/source \
		-e PACK=$@ \
		-e BEFORE_BUILD=before_build.sh \
		-e SOURCE=/source \
		-e TARGETS="windows/amd64 windows/386 darwin/amd64" \
		-e BUILDID=${BUILDID} \
		tudorg/beats-builder \
		github.com/elastic/beats
	# linux builds on debian 6
	docker run --rm \
		-v $(abspath build):/build \
		-v $(abspath xgo-scripts):/scripts \
		-v $(abspath ../..):/source \
		-e PACK=$@ \
		-e BEFORE_BUILD=before_build.sh \
		-e SOURCE=/source \
		-e TARGETS="linux/amd64 linux/386" \
		-e BUILDID=${BUILDID} \
		tudorg/beats-builder-deb6 \
		github.com/elastic/beats

.PHONY: filebeat
filebeat:
	@echo Cross-compiling $@
	# pure go cross compiling doesn't need C compilers but
	# we do it using the same docker instances for simplicity
	docker run --rm \
		-v $(abspath build):/build \
		-v $(abspath xgo-scripts):/scripts \
		-v $(abspath ../..):/source \
		-e PUREGO="yes" \
		-e PACK=$@ \
		-e BEFORE_BUILD=before_build.sh \
		-e SOURCE=/source \
		-e TARGETS="linux/amd64 linux/386 windows/amd64 windows/386 darwin/amd64" \
		-e BUILDID=${BUILDID} \
		tudorg/beats-builder \
		github.com/elastic/beats

.PHONY: winlogbeat
winlogbeat:
	@echo Cross-compiling $@
	# Winlogbeat is pure go and only needs Windows
	docker run --rm \
		-v $(abspath build):/build \
		-v $(abspath xgo-scripts):/scripts \
		-v $(abspath ../..):/source \
		-e PUREGO="yes" \
		-e PACK=$@ \
		-e BEFORE_BUILD=before_build.sh \
		-e SOURCE=/source \
		-e TARGETS="windows/amd64 windows/386" \
		-e BUILDID=${BUILDID} \
		tudorg/beats-builder \
		github.com/elastic/beats

include ./scripts/Makefile

