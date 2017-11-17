BEAT_NAME=winlogbeat
BEAT_TITLE=Winlogbeat
BEAT_DESCRIPTION=Winlogbeat ships Windows event logs to Elasticsearch or Logstash.
SYSTEM_TESTS=true
TEST_ENVIRONMENT=false

GOX_OS=windows
TARGETS="windows/amd64 windows/386"
PACKAGES=winlogbeat/win

CGO=false		# don't need Cgo in Winlogbeat

include ../libbeat/scripts/Makefile

.PHONY: gen
gen:
	GOOS=windows GOARCH=386 go generate -v -x ./...

# This is called by the beats packer before building starts
.PHONY: before-build
before-build:

# Collects all dependencies and then calls update
.PHONY: collect
collect:
