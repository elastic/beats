BUILDID=$(shell git rev-parse HEAD)

.PHONY: all
all: {beat}/deb {beat}/rpm {beat}/darwin {beat}/win {beat}/bin \
	build/upload/build_id.txt

ES_BEATS?=../../vendor/github.com/elastic/beats
include $(ES_BEATS)/dev-tools/packer/scripts/Makefile


.PHONY: {beat}
{beat}: build/upload
	# cross compile on ubuntu
	docker run --rm \
		-v $(abspath build):/build \
		-v $(abspath $(ES_BEATS)/dev-tools/packer/xgo-scripts):/scripts \
		-v $(abspath ../..):/source \
		-e PACK=$@ \
		-e BEFORE_BUILD=before_build.sh \
		-e SOURCE=/source \
		-e BUILDID=${BUILDID} \
		tudorg/beats-builder \
		{beat_path}
	# linux builds on debian 6
	docker run --rm \
		-v $(abspath build):/build \
		-v $(abspath $(ES_BEATS)/dev-tools/packer/xgo-scripts):/scripts \
		-v $(abspath ../..):/source \
		-e PACK=$@ \
		-e BEFORE_BUILD=before_build.sh \
		-e SOURCE=/source \
		-e BUILDID=${BUILDID} \
		tudorg/beats-builder-deb6 \
		{beat_path}
