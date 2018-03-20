BEAT_NAME={beat}
BEAT_PATH={beat_path}
BEAT_URL=https://${BEAT_PATH}
BEAT_DOC_URL=https://path/to/{beat}/doc
BEAT_DESCRIPTION=My description
BEAT_LICENSE=My license
COPYRIGHT_TXT=My company
COPYRIGHT_YEAR={current_year}

ES_BEATS=./_beats
PREFIX=.
BEAT_GOPATH=$(firstword $(subst :, ,${GOPATH}))
BEATS_VERSION?=master # For specific version: BEATS_VERSION?=6.1
NOTICE_FILE=NOTICE
LICENSE_FILE=LICENSE.txt

NOW=$(shell date -u '+%Y-%m-%dT%H:%M:%S')
GOBUILD_FLAGS=-i -ldflags "-X $(BEAT_PATH)/vendor/github.com/elastic/beats/libbeat/version.buildTime=$(NOW) -X $(BEAT_PATH)/vendor/github.com/elastic/beats/libbeat/version.commit=$(COMMIT_ID)"
GOPACKAGES=$(shell govendor list -no-status +local)

SYSTEM_TESTS=false
TEST_ENVIRONMENT=true
TESTING_ENVIRONMENT=snapshot-noxpack

# Path to the libbeat Makefile
-include $(ES_BEATS)/libbeat/scripts/Makefile

setup:
	$(MAKE) update-dependencies
	git init

# updates {beat} dependencies in the vendor folder
vendor:
	mkdir -p vendor
	govendor init

update-dependencies: vendor
	$(MAKE) update-libbeat
	$(MAKE) update-{beat}-dependencies

update-libbeat:
	rm -rf vendor/github.com/elastic/beats
	# Fetch libbeat dependency to the vendor folder
	govendor fetch github.com/elastic/beats/...@$(BEATS_VERSION)
	govendor fetch github.com/elastic/beats/libbeat/kibana/@$(BEATS_VERSION)
	# Change the origin of all libbeat's dependencies to elastic/beats git repo
	python scripts/patch_vendorjson.py
	# Update all dependencies with the new orign...
	BEATS_VERSION=$(BEATS_VERSION) /bin/bash scripts/update_beats.sh
	# Change the origin of all additional libbeat's dependencies to elastic/beats git repo
	python scripts/patch_vendorjson.py --done
	# Update all libbeat's dependencies one more time with the new origin...
	BEATS_VERSION=$(BEATS_VERSION) /bin/bash scripts/update_beats.sh
	# Update documentation and configuration files
	$(MAKE) update
	@echo --- Use this commit message: Update beats framework to `cat vendor/vendor.json | python -c 'import sys, json; print([p["revision"] for p in json.load(sys.stdin)["package"] if p["path"] == "github.com/elastic/beats/libbeat/beat"][0][:7])'`

update-{beat}-dependencies:
	# List all golang packages that are imported by {beat} in addition to libbeat
	#govendor fetch github.com/one/package
	#govendor fetch github.com/another/package

# This is called by the beats packer before building starts
.PHONY: before-build
before-build:

# Collects all dependencies and then calls update
.PHONY: collect
collect: notice

.PHONY: check-full
check-full: check
	@# Validate that all updates were committed
	@$(MAKE) update
	@$(MAKE) check
	@git diff | cat
	@git update-index --refresh
	@git diff-index --exit-code HEAD --

.PHONY: notice
notice: python-env
	@echo "Generating NOTICE.txt"
	$(PYTHON_ENV)/bin/python ${ES_BEATS}/dev-tools/generate_notice.py  -s "./vendor/github.com/elastic/beats" -b "$(BEAT_NAME)" -c "$(COPYRIGHT_TXT)" .
