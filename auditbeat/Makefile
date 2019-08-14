BEAT_NAME=auditbeat
BEAT_TITLE=Auditbeat
SYSTEM_TESTS=true
TEST_ENVIRONMENT?=true
GOX_OS?=linux windows
ES_BEATS?=..
EXCLUDE_COMMON_UPDATE_TARGET=true

include ${ES_BEATS}/libbeat/scripts/Makefile

.PHONY: update
update: mage
	mage update
