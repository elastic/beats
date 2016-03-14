DATE:=$(shell date +%y%m%d%H%M%S)
BUILDID?=$(DATE)


.PHONY: all
all: packetbeat/deb packetbeat/rpm packetbeat/darwin packetbeat/win packetbeat/bin \
	topbeat/deb topbeat/rpm topbeat/darwin topbeat/win topbeat/bin \
	filebeat/deb filebeat/rpm filebeat/darwin filebeat/win filebeat/bin \
	metricbeat/deb metricbeat/rpm metricbeat/darwin metricbeat/win metricbeat/bin \
	winlogbeat/win \
	build/upload/build_id.txt latest

.PHONY: packetbeat topbeat filebeat metricbeat winlogbeat
packetbeat topbeat filebeat metricbeat winlogbeat: build/upload
	# cross compile on ubuntu
	cd build && xgo -image=tudorg/beats-builder \
		-before-build=../xgo-scripts/before_build.sh \
		-pkg $@ \
		-source ../../.. \
		github.com/elastic/beats
	# linux builds on debian 6
	cd build && xgo -image=tudorg/beats-builder-deb6 \
		-before-build=../xgo-scripts/before_build.sh \
		-pkg $@ \
		-source ../../.. \
		github.com/elastic/beats

include ./scripts/Makefile

