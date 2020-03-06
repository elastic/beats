ES_BEATS ?= ../..

include $(ES_BEATS)/dev-tools/make/xpack.mk

# Builds the environment to test beat
.PHONY: build-image
build-image: write-environment
	${DOCKER_COMPOSE} build ${DOCKER_NOCACHE} --force-rm
