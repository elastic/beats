BEAT_NAME?=beatless
LICENSE=Elastic
BEAT_TITLE?=Beatless
SYSTEM_TESTS?=true
BEAT_PATH?=github.com/elastic/beats/x-pack/${BEAT_NAME}
TEST_ENVIRONMENT?=false
GOX_FLAGS=-arch="amd64 386 arm ppc64 ppc64le"
ES_BEATS?=../../
FIELDS_FILE_PATH=module

# Path to the libbeat Makefile
include $(ES_BEATS)/libbeat/scripts/Makefile

# Runs all collection steps and updates afterwards
.PHONY: collect
collect:
