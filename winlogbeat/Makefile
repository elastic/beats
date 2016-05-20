BEATNAME=winlogbeat
SYSTEM_TESTS=true
TEST_ENVIRONMENT=false
GOX_OS=windows

include ../libbeat/scripts/Makefile

.PHONY: gen
gen:
	GOOS=windows GOARCH=386 go generate -v -x ./...

# This is called by the beats packer before building starts
.PHONY: before-build
before-build:
