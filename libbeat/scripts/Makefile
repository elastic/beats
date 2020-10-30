### VARIABLE SETUP ###
### Application using libbeat may override the following variables in their Makefile
BEAT_NAME?=libbeat## @packaging Name of the binary
LICENSE?=ASL2
BEAT_TITLE?=${BEAT_NAME}## @packaging Title of the application
BEAT_PATH?=github.com/elastic/beats/${BEAT_NAME}
BEAT_PACKAGE_NAME?=${BEAT_NAME}
BEAT_INDEX_PREFIX?=${BEAT_NAME}
BEAT_URL?=https://www.elastic.co/products/beats/${BEAT_NAME} ## @packaging Link to the homepage of the application
BEAT_DOC_URL?=https://www.elastic.co/guide/en/beats/${BEAT_NAME}/current/index.html ## @packaging Link to the user documentation of the application
BEAT_LICENSE?=ASL 2.0 ## @packaging Software license of the application
BEAT_VENDOR?=Elastic ## @packaging Name of the vendor of the application
BEAT_GOPATH=$(firstword $(subst :, ,${GOPATH}))
BEAT_REF_YAML?=true
ES_BEATS?=..## @community_beat Must be set to ./vendor/github.com/elastic/beats. It must always be a relative path.
GOPACKAGES?=$(shell go list ${BEAT_PATH}/... | grep -v /vendor/ | grep -v /scripts/cmd/ )
PACKER_TEMPLATES_DIR?=${ES_BEATS}/dev-tools/packer ## @Building Directory of templates that are used by "make release"
NOTICE_FILE?=../NOTICE.txt
LICENSE_FILE?=../licenses/APACHE-LICENSE-2.0.txt
ELASTIC_LICENSE_FILE?=../licenses/ELASTIC-LICENSE.txt
SECCOMP_BINARY?=${BEAT_NAME}
SECCOMP_BLACKLIST?=${ES_BEATS}/libbeat/common/seccomp/seccomp-profiler-blacklist.txt
SECCOMP_ALLOWLIST?=${ES_BEATS}/libbeat/common/seccomp/seccomp-profiler-allow.txt
MAGE_PRESENT := $(shell command -v mage 2> /dev/null)
MAGE_IMPORT_PATH?=github.com/elastic/beats/vendor/github.com/magefile/mage
export MAGE_IMPORT_PATH

space:=$() #
comma:=,

ARCH?=$(shell uname -m)
# Hidden directory to install dependencies for jenkins
export PATH := ./bin:$(PATH)
GOFILES = $(shell find . -type f -name '*.go')
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "*/vendor/*")
GOFILES_ALL = $(GOFILES) $(shell find $(ES_BEATS) -type f -name '*.go')
GOPACKAGES_STRESSTESTS=$(shell find . -name '*.go' | xargs awk 'FNR>1 {nextfile} /\+build.*stresstest/ {print FILENAME; nextfile}' | xargs dirname | uniq)
SHELL=bash
ES_HOST?="elasticsearch"
PWD=$(shell pwd)
BUILD_DIR?=$(shell pwd)/build
PKG_BUILD_DIR?=$(BUILD_DIR)/package${PKG_SUFFIX}
PKG_UPLOAD_DIR?=$(BUILD_DIR)/upload
COVERAGE_DIR?=${BUILD_DIR}/coverage
COVERAGE_TOOL?=${BEAT_GOPATH}/bin/gotestcover
COVERAGE_TOOL_REPO?=github.com/elastic/beats/vendor/github.com/pierrre/gotestcover
TESTIFY_TOOL_REPO?=github.com/elastic/beats/vendor/github.com/stretchr/testify/assert
NOW=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GOBUILD_FLAGS?=-i -ldflags "-X github.com/elastic/beats/libbeat/version.buildTime=$(NOW) -X github.com/elastic/beats/libbeat/version.commit=$(COMMIT_ID)"
GOIMPORTS=goimports
GOIMPORTS_REPO?=github.com/elastic/beats/vendor/golang.org/x/tools/cmd/goimports
GOIMPORTS_LOCAL_PREFIX?=github.com/elastic
GOLINT=golint
GOLINT_REPO?=github.com/golang/lint/golint
REVIEWDOG?=reviewdog -conf ${ES_BEATS}/reviewdog.yml
REVIEWDOG_OPTIONS?=-diff "git diff master"
REVIEWDOG_REPO?=github.com/reviewdog/reviewdog/cmd/reviewdog
PROCESSES?= 4
TIMEOUT?= 90
PYTHON_TEST_FILES?=$(shell find . -type f -name 'test_*.py' -not -path "*/build/*" -not -path "*/vendor/*")
NOSETESTS_OPTIONS?=--process-timeout=$(TIMEOUT) --with-timer -v --with-xunit --xunit-file=${BUILD_DIR}/TEST-system.xml ## @testing the options to pass when calling nosetests
TEST_ENVIRONMENT?=false ## @testing if true, "make testsuite" runs integration tests and system tests in a dockerized test environment
SYSTEM_TESTS?=false ## @testing if true, "make test" and "make testsuite" run unit tests and system tests
STRESS_TESTS?=false ## @testing if true, "make test" and "make testsuite" run also run the stress tests
STRESS_TEST_OPTIONS?=-timeout=20m -race -v
GOX_OS?=linux darwin windows freebsd netbsd openbsd ## @Building List of all OS to be supported by "make crosscompile".
GOX_OSARCH?=!darwin/arm !darwin/arm64 !darwin/386 ## @building Space separated list of GOOS/GOARCH pairs to build by "make crosscompile".
GOX_FLAGS?= ## @building Additional flags to append to the gox command used by "make crosscompile".
# XXX: Should be switched back to `snapshot` once the Elasticsearch
# snapshots are working. https://github.com/elastic/beats/pull/6416
TESTING_ENVIRONMENT?=snapshot## @testing The name of the environment under test
BEAT_VERSION=$(shell head -n 1 ${ES_BEATS}/libbeat/docs/version.asciidoc | cut -c 17- )
COMMIT_ID=$(shell git rev-parse HEAD)
DOCKER_COMPOSE_PROJECT_NAME?=${BEAT_NAME}${TESTING_ENVIRONMENT//-}${BEAT_VERSION//-}${COMMIT_ID} ## @testing The name of the docker-compose project used by the integration and system tests
DOCKER_COMPOSE?=TESTING_ENVIRONMENT=${TESTING_ENVIRONMENT} ES_BEATS=${ES_BEATS} docker-compose -p ${DOCKER_COMPOSE_PROJECT_NAME} -f docker-compose.yml
DOCKER_CACHE?=1 ## @miscellaneous If set to 0, all docker images are created without cache
GOPACKAGES_COMMA_SEP=$(subst $(space),$(comma),$(strip ${GOPACKAGES}))
PYTHON_ENV?=${BUILD_DIR}/python-env
PIP_INSTALL_PARAMS?=
BUILDID?=$(shell git rev-parse HEAD) ## @Building The build ID
VIRTUALENV_PARAMS?=
INTEGRATION_TESTS?=
FIND?=. ${PYTHON_ENV}/bin/activate; find . -type f -not -path "*/vendor/*" -not -path "*/build/*" -not -path "*/.git/*"
PERM_EXEC?=$(shell [ `uname -s` = "Darwin" ] && echo "+111" || echo "/a+x")
XPACK_ONLY?=false
XPACK_DIR=${ES_BEATS}/x-pack/${BEAT_NAME}

ifeq ($(DOCKER_CACHE),0)
	DOCKER_NOCACHE=--no-cache
endif

# Conditionally enable the race detector when RACE_DETECTOR=1.
ifeq ($(RACE_DETECTOR),1)
	RACE=-race
endif

### BUILDING ###


${BEAT_NAME}: $(GOFILES_ALL) ## @build build the beat application
	go build $(GOBUILD_FLAGS)

# Create test coverage binary
${BEAT_NAME}.test: $(GOFILES_ALL)
	@go build -i -o /dev/null
	@go test $(RACE) -c -coverpkg ${GOPACKAGES_COMMA_SEP}

.PHONY: crosscompile
crosscompile: ## @build Cross-compile beat for the OS'es specified in GOX_OS variable. The binaries are placed in the build/bin directory.
crosscompile: $(GOFILES)
ifneq ($(shell [[ $(BEAT_NAME) == journalbeat ]] && echo true ),true)
	go get github.com/mitchellh/gox
	mkdir -p ${BUILD_DIR}/bin
	gox -output="${BUILD_DIR}/bin/{{.Dir}}-{{.OS}}-{{.Arch}}" -os="$(strip $(GOX_OS))" -osarch="$(strip $(GOX_OSARCH))" ${GOX_FLAGS}
endif

.PHONY: check
check: check-headers python-env prepare-tests ## @build Checks project and source code if everything is according to standard
	@go vet ${GOPACKAGES}
	@go get $(GOIMPORTS_REPO)
	@goimports -local ${GOIMPORTS_LOCAL_PREFIX} -l ${GOFILES_NOVENDOR} | (! grep .) || (echo "Code differs from goimports' style ^" && false)
	@${FIND} -name '*.py' -exec autopep8 -d --max-line-length 120  {} \; | (! grep . -q) || (echo "Code differs from autopep8's style" && false)
	@${FIND} -wholename "*tests/system/test_*.py" -perm ${PERM_EXEC} -exec false {} + || (echo "Python test files shouldn't be executable, otherwise nose doesn't find them" && false)
	@${FIND} -name "*.yml" -perm ${PERM_EXEC} -exec false {} + || (echo "YAML files should not be executable" && false)

.PHONY: check-headers
check-headers:
ifndef CHECK_HEADERS_DISABLED
	@go get github.com/elastic/go-licenser
	@go-licenser -d -license ${LICENSE}
endif

.PHONY: add-headers
add-headers:
ifndef CHECK_HEADERS_DISABLED
	@go get github.com/elastic/go-licenser
	@go-licenser -license ${LICENSE}
endif

.PHONY: fmt
fmt: add-headers python-env ## @build Runs `goimports -l -w` and `autopep8`on the project's source code, modifying any files that do not match its style.
	@go get $(GOIMPORTS_REPO)
	@goimports -local ${GOIMPORTS_LOCAL_PREFIX} -l -w ${GOFILES_NOVENDOR}
	@${FIND} -name '*.py' -exec ${PYTHON_ENV}/bin/autopep8 --in-place --max-line-length 120  {} \;

.PHONY: lint
lint:
	@go get $(GOLINT_REPO) $(REVIEWDOG_REPO)
	$(REVIEWDOG) $(REVIEWDOG_OPTIONS)

.PHONY: clean
clean:  ## @build  Cleans up all files generated by the build steps
	@rm -rf build
	@rm -f docker-compose.yml.lock
	@rm -f ${BEAT_NAME} ${BEAT_NAME}.test ${BEAT_NAME}.exe ${BEAT_NAME}.test.exe
	@rm -f _meta/fields.generated.yml fields.yml
	@rm -rf $(PWD)/_meta/kibana.generated
	@rm -f ${BEAT_NAME}.template*.json
	@-mage -clean 2> /dev/null

.PHONY: ci
ci:  ## @build Shortcut for continuous integration. This should always run before merging.
	$(MAKE)
	$(MAKE) check
	$(MAKE) testsuite

### Testing ###
# Unless stated otherwise, all tests are always run with coverage reporting enabled.

# Preparation for tests
.PHONY: prepare-tests
prepare-tests:
	mkdir -p ${COVERAGE_DIR}
	# gotestcover is needed to fetch coverage for multiple packages
	go get ${COVERAGE_TOOL_REPO}
	# testify is needed for unit and integration tests
	go get ${TESTIFY_TOOL_REPO}

.PHONY: unit-tests
unit-tests: ## @testing Runs the unit tests with coverage.  Race is not enabled for unit tests because tests run much slower.
unit-tests: prepare-tests
	go test -i ${GOPACKAGES}
	$(COVERAGE_TOOL) $(RACE) -coverprofile=${COVERAGE_DIR}/unit.cov  ${GOPACKAGES}

.PHONY: unit
unit: ## @testing Runs the unit tests without coverage reports.
	go test -i ${GOPACKAGES}
	go test $(RACE) ${GOPACKAGES}

.PHONY: integration-tests
integration-tests: ## @testing Run integration tests. Unit tests are run as part of the integration tests.
integration-tests: prepare-tests
	rm -f docker-compose.yml.lock
	go test -i ${GOPACKAGES}
	$(COVERAGE_TOOL) -tags=integration $(RACE) -coverprofile=${COVERAGE_DIR}/integration.cov ${GOPACKAGES}

#
.PHONY: integration-tests-environment
integration-tests-environment:  ## @testing Runs the integration inside a virtual environment. This can be run on any docker-machine (local, remote)
integration-tests-environment: prepare-tests build-image
	# If you want to get logs on CI when the `docker-compose` fails, you can
	# add  `|| ${DOCKER_COMPOSE} logs --tail 200` after the command.
	#
	# This will make docker-compose command to display the logs on stdout on error, It's not enabled
	# by default because it can create noise if the test inside the container fails.
	${DOCKER_COMPOSE} run beat make integration-tests RACE_DETECTOR=$(RACE_DETECTOR) DOCKER_COMPOSE_PROJECT_NAME=${DOCKER_COMPOSE_PROJECT_NAME}

# Runs the system tests
.PHONY: system-tests
system-tests: ## @testing Runs the system tests
system-tests: prepare-tests ${BEAT_NAME}.test python-env
	. ${PYTHON_ENV}/bin/activate; INTEGRATION_TESTS=${INTEGRATION_TESTS} TESTING_ENVIRONMENT=${TESTING_ENVIRONMENT} DOCKER_COMPOSE_PROJECT_NAME=${DOCKER_COMPOSE_PROJECT_NAME} nosetests ${PYTHON_TEST_FILES} ${NOSETESTS_OPTIONS}
	python ${ES_BEATS}/dev-tools/aggregate_coverage.py -o ${COVERAGE_DIR}/system.cov ${BUILD_DIR}/system-tests/run

# Runs the system tests
.PHONY: system-tests-environment
system-tests-environment:  ## @testing Runs the system tests inside a virtual environment. This can be run on any docker-machine (local, remote)
system-tests-environment: prepare-tests build-image
	${DOCKER_COMPOSE} run -e INTEGRATION_TESTS=1 -e TESTING_ENVIRONMENT=${TESTING_ENVIRONMENT} -e DOCKER_COMPOSE_PROJECT_NAME=${DOCKER_COMPOSE_PROJECT_NAME} beat make system-tests

.PHONY: fast-system-tests
fast-system-tests: ## @testing Runs system tests without coverage reports and in parallel
fast-system-tests: ${BEAT_NAME}.test python-env
	. ${PYTHON_ENV}/bin/activate; nosetests ${PYTHON_TEST_FILES} ${NOSETESTS_OPTIONS}

# Runs the go based stress tests
.PHONY: stress-tests
stress-tests: ## @testing Runs the stress tests with race detector enabled
stress-tests:
	if [ -n '${GOPACKAGES_STRESSTESTS}' ]; then \
		go test --tags=stresstest ${STRESS_TEST_OPTIONS} ${GOPACKAGES_STRESSTESTS}; \
	fi

# Run benchmark tests
.PHONY: benchmark-tests
benchmark-tests: ## @testing Runs benchmarks (NOT YET IMPLEMENTED)
	# No benchmark tests exist so far
	#go test -bench=. ${GOPACKAGES}

# Run load tests
.PHONY: load-tests
load-tests: ## @testing Runs load tests
	. ${PYTHON_ENV}/bin/activate; LOAD_TESTS=1  nosetests ${PYTHON_TEST_FILES} --processes=$(PROCESSES) --process-timeout=$(TIMEOUT) -a 'load'

# Sets up the virtual python environment
.PHONY: python-env
python-env: ${ES_BEATS}/libbeat/tests/system/requirements.txt
	@test -d ${PYTHON_ENV} || virtualenv $(if ${PYTHON_EXE},-p ${PYTHON_EXE}) ${VIRTUALENV_PARAMS} ${PYTHON_ENV}
	@. ${PYTHON_ENV}/bin/activate && pip install ${PIP_INSTALL_PARAMS} -q --upgrade pip ; \
	if [ -a ./tests/system/requirements.txt ] && [ ! ${ES_BEATS}/libbeat/tests/system/requirements.txt -ef ./tests/system/requirements.txt ] ; then \
		. ${PYTHON_ENV}/bin/activate && pip install ${PIP_INSTALL_PARAMS} -qUr ${ES_BEATS}/libbeat/tests/system/requirements.txt -Ur ./tests/system/requirements.txt ; \
	else \
		. ${PYTHON_ENV}/bin/activate && pip install ${PIP_INSTALL_PARAMS} -qUr ${ES_BEATS}/libbeat/tests/system/requirements.txt ; \
	fi
	@# Work around pip bug. See: https://github.com/pypa/pip/issues/4464
	@find ${PYTHON_ENV} -type d -name 'dist-packages' -exec sh -c "echo dist-packages > {}.pth" ';'

.PHONY: test
test: ## @testing Runs unit and system tests without coverage reports
test: unit
	if [ $(STRESS_TESTS) = true ]; then \
		 $(MAKE) stress-tests; \
	fi

	if [ $(SYSTEM_TESTS) = true ]; then \
		 $(MAKE) fast-system-tests; \
	fi

.PHONY: testsuite
testsuite: ## @testing Runs all tests and generates the coverage reports
testsuite: clean update
	$(MAKE) unit-tests

	if [ $(STRESS_TESTS) = true ]; then \
		 $(MAKE) stress-tests; \
	fi

	# Setups environment if TEST_ENVIRONMENT is set to true
	# Only runs integration tests with test environment
	if [ $(TEST_ENVIRONMENT) = true ]; then \
		 $(MAKE) integration-tests-environment; \
	fi

	# Runs system and system integration tests if SYSTEM_TESTS is set to true
	if [ $(SYSTEM_TESTS) = true ]; then \
		if [ $(TEST_ENVIRONMENT) = true ]; then \
			$(MAKE) system-tests-environment; \
		else \
			$(MAKE) system-tests; \
		fi \
	fi

	if [ $(TEST_ENVIRONMENT) = true ]; then \
		$(MAKE) fix-permissions; \
	fi

	$(MAKE) benchmark-tests
	$(MAKE) coverage-report

	if [ $(TEST_ENVIRONMENT) = true ]; then \
		$(MAKE) stop-environment; \
    fi

# Generates a coverage report from the existing coverage files
.PHONY: coverage-report
coverage-report:
	python ${ES_BEATS}/dev-tools/aggregate_coverage.py -o ${COVERAGE_DIR}/full.cov ${COVERAGE_DIR}
	go tool cover -html=${COVERAGE_DIR}/full.cov -o ${COVERAGE_DIR}/full.html
	test ! -s ${COVERAGE_DIR}/integration.cov   || go tool cover -html=${COVERAGE_DIR}/integration.cov   -o ${COVERAGE_DIR}/integration.html
	test ! -s ${COVERAGE_DIR}/system.cov || go tool cover -html=${COVERAGE_DIR}/system.cov -o ${COVERAGE_DIR}/system.html
	test ! -s ${COVERAGE_DIR}/unit.cov   || go tool cover -html=${COVERAGE_DIR}/unit.cov   -o ${COVERAGE_DIR}/unit.html


.PHONY: fields
fields: mage
	@mage fields

ifndef EXCLUDE_COMMON_UPDATE_TARGET
.PHONY: update
update: python-env fields collect ## @build Update expects the most recent version of libbeat in the GOPATH
	@echo "Updating generated files for ${BEAT_NAME}"

	@mkdir -p _meta
	@# Update config files.
	@cat _meta/beat.yml ${ES_BEATS}/libbeat/_meta/config.yml | sed -e "s/beatname/${BEAT_NAME}/g;s/beat-index-prefix/${BEAT_INDEX_PREFIX}/g" > ${BEAT_NAME}.yml
ifneq (,$(wildcard _meta/beat.docker.yml))
	@cat _meta/beat.docker.yml ${ES_BEATS}/libbeat/_meta/config.docker.yml > ${BEAT_NAME}.docker.yml
	@chmod 0640 ${BEAT_NAME}.yml ${BEAT_NAME}.docker.yml
endif

	@# Update reference config files.
ifeq ($(BEAT_REF_YAML),true)
	@if [ -e _meta/beat.reference.yml ]; then \
      cat _meta/beat.reference.yml ${ES_BEATS}/libbeat/_meta/config.reference.yml | sed -e "s/beatname/${BEAT_NAME}/g;s/beat-index-prefix/${BEAT_INDEX_PREFIX}/g" > ${BEAT_NAME}.reference.yml ; \
  else \
      cat _meta/beat.yml ${ES_BEATS}/libbeat/_meta/config.reference.yml | sed -e "s/beatname/${BEAT_NAME}/g;s/beat-index-prefix/${BEAT_INDEX_PREFIX}/g" > ${BEAT_NAME}.reference.yml ; \
  fi;
	@chmod 0640 ${BEAT_NAME}.reference.yml
endif

ifneq ($(shell [[ $(BEAT_NAME) == libbeat || $(BEAT_NAME) == metricbeat ]] && echo true ),true)
	mkdir -p include
	go run  ${ES_BEATS}/dev-tools/cmd/asset/asset.go -license $(LICENSE) -pkg include -in fields.yml -out include/fields.go $(BEAT_NAME)
endif

	@# Update docs
	@mkdir -p docs
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/libbeat/scripts/generate_fields_docs.py $(PWD) ${BEAT_TITLE} ${ES_BEATS}

	@mkdir -p $(PWD)/_meta/kibana.generated
	@# Generate Kibana index pattern and copy dashboard files
	if [ -d $(PWD)/_meta/kibana ]; then \
		cp -pr $(PWD)/_meta/kibana/* $(PWD)/_meta/kibana.generated ; \
	fi
	@# Convert all dashboards to string
	@python ${ES_BEATS}/libbeat/scripts/unpack_dashboards.py --glob="./_meta/kibana.generated/6/dashboard/*.json"
	@mkdir -p $(PWD)/_meta/kibana.generated/5/index-pattern
	@mkdir -p $(PWD)/_meta/kibana.generated/6/index-pattern
	@go run ${ES_BEATS}/dev-tools/cmd/kibana_index_pattern/kibana_index_pattern.go \
	  -index '${BEAT_INDEX_PREFIX}-*' -beat ${BEAT_NAME} -fields $(PWD)/fields.yml \
	  -version ${BEAT_VERSION} -out _meta/kibana.generated
endif

.PHONY: docs
docs:  ## @build Builds the documents for the beat
	@if [ -d $(XPACK_DIR) ]; then \
		sh ${ES_BEATS}/script/build_docs.sh ${BEAT_NAME} ${BEAT_PATH}/docs ${BUILD_DIR} ${XPACK_DIR}; \
	else \
		sh ${ES_BEATS}/script/build_docs.sh ${BEAT_NAME} ${BEAT_PATH}/docs ${BUILD_DIR}; \
	fi

.PHONY: docs-preview
docs-preview:  ## @build Preview the documents for the beat in the browser
	PREVIEW=1 $(MAKE) docs

### SETUP commands ###
ES_URL?=http://localhost:9200
KIBANA_URL?=http://localhost:5601

.PHONY: import-dashboards
import-dashboards: update ${BEAT_NAME}
	${BEAT_GOPATH}/src/${BEAT_PATH}/${BEAT_NAME} setup -E setup.dashboards.directory=${PWD}/_meta/kibana.generated -E setup.kibana.host=${KIBANA_URL} --dashboards

### CONTAINER ENVIRONMENT ####

# Builds the environment to test beat
.PHONY: build-image
build-image: write-environment
	${DOCKER_COMPOSE} build ${DOCKER_NOCACHE} --pull --force-rm

# Runs the environment so the redis and elasticsearch can also be used for local development
# To use it for running the test, set ES_HOST and REDIS_HOST environment variable to the ip of your docker-machine.
.PHONY: start-environment
start-environment: stop-environment
	${DOCKER_COMPOSE} pull --include-deps
	${DOCKER_COMPOSE} up -d

.PHONY: stop-environment
stop-environment:
	-${DOCKER_COMPOSE} down -v

.PHONY: write-environment
write-environment:
	mkdir -p ${BUILD_DIR}
	echo "BEAT_STRICT_PERMS=false" > ${BUILD_DIR}/test.env
	echo "ES_HOST=${ES_HOST}" >> ${BUILD_DIR}/test.env
	echo "ES_PORT=9200" >> ${BUILD_DIR}/test.env
	echo "ES_USER=beats" >> ${BUILD_DIR}/test.env
	echo "ES_PASS=testing" >> ${BUILD_DIR}/test.env

# Tails the environment logs
.PHONY: env-logs
env-logs:
	${DOCKER_COMPOSE} logs -f

fix-permissions:
	# Change ownership of all files inside /build folder from root/root to current user/group
	docker run -v ${PWD}:/beat alpine:3.4 sh -c "find /beat -user 0 -exec chown -h $(shell id -u):$(shell id -g) {} \;"

set_version: ## @packaging VERSION=x.y.z set the version of the beat to x.y.z
	${ES_BEATS}/dev-tools/set_version ${VERSION}

get_version: ## @packaging get the version of the beat.
	@${ES_BEATS}/dev-tools/get_version

help: ## @help Show this help.
	@python ${ES_BEATS}/libbeat/scripts/generate_makefile_doc.py $(MAKEFILE_LIST)

help_variables: ## @help Show Makefile customizable variables.
	@python ${ES_BEATS}/libbeat/scripts/generate_makefile_doc.py --variables $(MAKEFILE_LIST)

# Generates a seccomp whitelist policy for the binary pointed to by
# SECCOMP_BINARY.
.PHONY: seccomp
seccomp:
	@go get github.com/elastic/beats/vendor/github.com/elastic/go-seccomp-bpf/cmd/seccomp-profiler
	@test -f ${SECCOMP_BINARY} || (echo "${SECCOMP_BINARY} binary is not built."; false)
	seccomp-profiler \
	-b "$(shell grep -v ^# "${SECCOMP_BLACKLIST}")" \
	-allow "$(shell grep -v ^# "${SECCOMP_ALLOWLIST}")" \
	-t "${ES_BEATS}/libbeat/common/seccomp/policy.go.tpl" \
	-pkg include \
	-out "include/seccomp_linux_{{.GOARCH}}.go" \
	${SECCOMP_BINARY}

# Generates seccomp profiles based on the binaries produced by the package target.
.PHONY: seccomp-package
seccomp-package:
	SECCOMP_BINARY=build/golang-crossbuild/${BEAT_NAME}-linux-386 $(MAKE) seccomp
	SECCOMP_BINARY=build/golang-crossbuild/${BEAT_NAME}-linux-amd64 $(MAKE) seccomp

### Packaging targets ####

.PHONY: mage
mage:
ifndef MAGE_PRESENT
	go install ${MAGE_IMPORT_PATH}
	@-mage -clean 2> /dev/null
endif

.PHONY: release
release: mage
	@mage package

.PHONY: package
snapshot: mage
	@SNAPSHOT=true mage package
