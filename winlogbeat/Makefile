BEAT_NAME=winlogbeat
BEAT_TITLE=Winlogbeat
SYSTEM_TESTS=true
TEST_ENVIRONMENT=false
ES_BEATS?=..
GOX_OS=windows
EXCLUDE_COMMON_UPDATE_TARGET=true

include ${ES_BEATS}/libbeat/scripts/Makefile

.PHONY: update
update: mage
	mage update

.PHONY: gen
gen:
	GOOS=windows GOARCH=386 go generate -v -x ./...

.PHONY: unit
unit: mage
	mage unitTest
