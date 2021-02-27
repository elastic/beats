BEAT_NAME=winlogbeat
BEAT_TITLE=Winlogbeat
SYSTEM_TESTS=true
TEST_ENVIRONMENT=false
GOX_OS=windows
EXCLUDE_COMMON_UPDATE_TARGET=true

include ../libbeat/scripts/Makefile

.PHONY: update
update: mage
	mage update

.PHONY: gen
gen:
	GOOS=windows GOARCH=386 go generate -v -x ./...
