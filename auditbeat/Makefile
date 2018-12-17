BEAT_NAME=auditbeat
BEAT_TITLE=Auditbeat
SYSTEM_TESTS=true
TEST_ENVIRONMENT?=true
GOX_OS?=linux windows ## @Building List of all OS to be supported by "make crosscompile".
ES_BEATS?=..
EXCLUDE_COMMON_UPDATE_TARGET=true

# Path to the libbeat Makefile
include ${ES_BEATS}/libbeat/scripts/Makefile

.PHONY: update
update: mage
	mage update
