BEAT_NAME?=functionbeat
LICENSE=Elastic
BEAT_TITLE?=Functionbeat
SYSTEM_TESTS?=true
BEAT_PATH?=github.com/elastic/beats/x-pack/${BEAT_NAME}
TEST_ENVIRONMENT?=true
GOX_FLAGS=-arch="amd64 386 arm ppc64 ppc64le"
ES_BEATS?=../../
FIELDS_FILE_PATH=module
XPACK_ONLY?=true
EXCLUDE_COMMON_UPDATE_TARGET=true

# Path to the libbeat Makefile
include $(ES_BEATS)/libbeat/scripts/Makefile

# Runs all collection steps and updates afterwards
.PHONY: collect
collect:

# Generate an artifact to be push on serverless provider.
.PHONY: linux
linux:
	GOOS=linux go build -o pkg/functionbeat
