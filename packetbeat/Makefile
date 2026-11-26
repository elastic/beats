BEAT_NAME?=packetbeat
BEAT_TITLE?=Packetbeat
SYSTEM_TESTS?=true
TEST_ENVIRONMENT?=true
ES_BEATS?=..
EXCLUDE_COMMON_UPDATE_TARGET=true

include ${ES_BEATS}/libbeat/scripts/Makefile

.PHONY: update
update: mage
	mage update

.PHONY: benchmark
benchmark:
	go test -short -bench=. ./... -cpu=2
