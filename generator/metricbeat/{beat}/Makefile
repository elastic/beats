BEAT_NAME={beat}
BEAT_PATH={beat_path}
BEAT_GOPATH=$(firstword $(subst :, ,${GOPATH}))
SYSTEM_TESTS=false
TEST_ENVIRONMENT=false
ES_BEATS?=./vendor/github.com/elastic/beats
GOPACKAGES=$(shell govendor list -no-status +local)
GOBUILD_FLAGS=-i -ldflags "-X $(BEAT_PATH)/vendor/github.com/elastic/beats/libbeat/version.buildTime=$(NOW) -X $(BEAT_PATH)/vendor/github.com/elastic/beats/libbeat/version.commit=$(COMMIT_ID)"
MAGE_IMPORT_PATH=${BEAT_PATH}/vendor/github.com/magefile/mage
CHECK_HEADERS_DISABLED=true

# Path to the libbeat Makefile
-include $(ES_BEATS)/metricbeat/Makefile

# Initial beat setup
.PHONY: setup
setup: copy-vendor git-init
	#call make recursively so we can reload the above include.
	#Only needed during the first setup phase, before /vendor exists
	$(MAKE) create-metricset update git-add

# Copy beats into vendor directory
.PHONY: copy-vendor
copy-vendor: vendor-check
	mkdir -p vendor/github.com/elastic/beats
	git archive --remote ${BEAT_GOPATH}/src/github.com/elastic/beats HEAD | tar -x --exclude=x-pack -C vendor/github.com/elastic/beats
	ln -sf ${PWD}/vendor/github.com/elastic/beats/metricbeat/scripts/generate_imports_helper.py ${PWD}/vendor/github.com/elastic/beats/script/generate_imports_helper.py
	mkdir -p vendor/github.com/magefile
	cp -R ${BEAT_GOPATH}/src/github.com/elastic/beats/vendor/github.com/magefile/mage vendor/github.com/magefile
	cp -R ${BEAT_GOPATH}/src/github.com/elastic/beats/vendor/github.com/pkg vendor/github.com/

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