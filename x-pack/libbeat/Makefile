BEAT_NAME=libbeat
ES_BEATS?=../..
XPACK_BEAT_PATH?=github.com/elastic/beats/x-pack/${BEAT_NAME}
GOPACKAGES?=$(shell go list ${BEAT_PATH}/... ${XPACK_BEAT_PATH}/... | grep -v /vendor/ | grep -v /scripts/cmd/ )
LICENSE=Elastic
TEST_ENVIRONMENT?=true
SYSTEM_TESTS=true

include ../../libbeat/scripts/Makefile

collect update fields: ;
