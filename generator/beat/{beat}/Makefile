BEAT_NAME={beat}
BEAT_PATH={beat_path}
BEAT_GOPATH=$(firstword $(subst :, ,${GOPATH}))
SYSTEM_TESTS=false
TEST_ENVIRONMENT=false
ES_BEATS?=./vendor/github.com/elastic/beats
LIBBEAT_MAKEFILE=$(ES_BEATS)/libbeat/scripts/Makefile
GOPACKAGES=$(shell govendor list -no-status +local)
GOBUILD_FLAGS=-i -ldflags "-X $(BEAT_PATH)/vendor/github.com/elastic/beats/libbeat/version.buildTime=$(NOW) -X $(BEAT_PATH)/vendor/github.com/elastic/beats/libbeat/version.commit=$(COMMIT_ID)"
MAGE_IMPORT_PATH=${BEAT_PATH}/vendor/github.com/magefile/mage
NO_COLLECT=true
CHECK_HEADERS_DISABLED=true

# Path to the libbeat Makefile
-include $(LIBBEAT_MAKEFILE)

# Initial beat setup
.PHONY: setup
setup: pre-setup git-add

pre-setup: copy-vendor git-init
	$(MAKE) -f $(LIBBEAT_MAKEFILE) mage ES_BEATS=$(ES_BEATS)
	$(MAKE) -f $(LIBBEAT_MAKEFILE) update BEAT_NAME=$(BEAT_NAME) ES_BEATS=$(ES_BEATS) NO_COLLECT=$(NO_COLLECT)

# Copy beats into vendor directory
.PHONY: copy-vendor
copy-vendor: vendor-check
	mkdir -p vendor/github.com/elastic/beats
	git archive --remote ${BEAT_GOPATH}/src/github.com/elastic/beats HEAD | tar -x --exclude=x-pack -C vendor/github.com/elastic/beats
	mkdir -p vendor/github.com/magefile
	cp -R vendor/github.com/elastic/beats/vendor/github.com/magefile/mage vendor/github.com/magefile

.PHONY: git-init
git-init:
	git init

.PHONY: git-add
git-add:
	git add -A
	git commit -q -m "Add generated {beat} files"


.PHONY: vendor-check
vendor-check:
	@if output=$$(git -C ${BEAT_GOPATH}/src/github.com/elastic/beats status --porcelain) && [ ! -z "$${output}" ]; then printf "\033[31mWARNING: elastic/beats has uncommitted changes, these will not be in the vendor directory!\033[0m\n"; fi