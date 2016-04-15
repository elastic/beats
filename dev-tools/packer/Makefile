BUILDID=$(shell git rev-parse HEAD)

.PHONY: all
all: packetbeat/deb packetbeat/rpm packetbeat/darwin packetbeat/win packetbeat/bin \
	topbeat/deb topbeat/rpm topbeat/darwin topbeat/win topbeat/bin \
	filebeat/deb filebeat/rpm filebeat/darwin filebeat/win filebeat/bin \
	metricbeat/deb metricbeat/rpm metricbeat/darwin metricbeat/win metricbeat/bin \
	winlogbeat/win \
	build/upload/build_id.txt

.PHONY: packetbeat topbeat filebeat metricbeat winlogbeat
packetbeat topbeat filebeat metricbeat winlogbeat: build/upload
	# cross compile on ubuntu
	docker run --rm \
		-v $(abspath build):/build \
		-v $(abspath xgo-scripts):/scripts \
		-v $(abspath ../..):/source \
		-e PACK=$@ \
		-e BEFORE_BUILD=before_build.sh \
		-e SOURCE=/source \
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
		-e BUILDID=${BUILDID} \
		tudorg/beats-builder-deb6 \
		github.com/elastic/beats

include ./scripts/Makefile

