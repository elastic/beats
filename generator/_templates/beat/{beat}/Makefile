BEAT_NAME={beat}
BEAT_PATH={beat_path}
BEAT_GOPATH=$(firstword $(subst :, ,${GOPATH}))
SYSTEM_TESTS=false
TEST_ENVIRONMENT=false
ES_BEATS_IMPORT_PATH=github.com/elastic/beats/v7
ES_BEATS?=./vendor/${ES_BEATS_IMPORT_PATH}
LIBBEAT_MAKEFILE=$(ES_BEATS)/libbeat/scripts/Makefile
GOPACKAGES=$(shell go list ${BEAT_PATH}/... | grep -v /tools)
GOBUILD_FLAGS=-i -ldflags "-X ${ES_BEATS_IMPORT_PATH}/libbeat/version.buildTime=$(NOW) -X ${ES_BEATS_IMPORT_PATH}/libbeat/version.commit=$(COMMIT_ID)"
MAGE_IMPORT_PATH=github.com/magefile/mage
NO_COLLECT=true
CHECK_HEADERS_DISABLED=true

# Path to the libbeat Makefile
-include $(LIBBEAT_MAKEFILE)

.PHONY: copy-vendor
copy-vendor:
	mage vendorUpdate
