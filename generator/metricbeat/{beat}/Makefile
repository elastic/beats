BEAT_NAME={beat}
BEAT_PATH={beat_path}
BEAT_URL=https://${BEAT_PATH}
SYSTEM_TESTS=false
TEST_ENVIRONMENT=false
ES_BEATS?=./vendor/github.com/elastic/beats
GOPACKAGES=$(shell govendor list -no-status +local)
PREFIX?=.
NOTICE_FILE=NOTICE

# Path to the libbeat Makefile
-include $(ES_BEATS)/metricbeat/Makefile

# Initial beat setup
.PHONY: setup
setup: copy-vendor
	$(MAKE) create-metricset
	$(MAKE) collect

# Copy beats into vendor directory
.PHONY: copy-vendor
copy-vendor:
	mkdir -p vendor/github.com/elastic/
	cp -R ${GOPATH}/src/github.com/elastic/beats vendor/github.com/elastic/
	rm -rf vendor/github.com/elastic/beats/.git vendor/github.com/elastic/beats/x-pack

# This is called by the beats packer before building starts
.PHONY: before-build
before-build:
