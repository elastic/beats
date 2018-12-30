BEAT_NAME=winlogbeat
BEAT_TITLE=Winlogbeat
SYSTEM_TESTS=true
TEST_ENVIRONMENT=false
GOX_OS=windows

include ../libbeat/scripts/Makefile

.PHONY: gen
gen:
	GOOS=windows GOARCH=386 go generate -v -x ./...

# Collects all dependencies and then calls update
.PHONY: collect
collect:
