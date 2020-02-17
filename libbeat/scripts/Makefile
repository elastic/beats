### VARIABLE SETUP ###
### Application using libbeat may override the following variables in their Makefile
BEAT_NAME?=libbeat## @packaging Name of the binary
BEAT_FULL_NAME?=${BEAT_NAME}
LICENSE?=ASL2
BEAT_TITLE?=${BEAT_NAME}## @packaging Title of the application
BEATS_ROOT?=github.com/elastic/beats
BEATS_ROOT_IMPORT_PATH?=${BEATS_ROOT}/v7
BEAT_PATH?=${BEATS_ROOT}/${BEAT_NAME}
BEAT_IMPORT_PATH?=${BEATS_ROOT_IMPORT_PATH}/${BEAT_FULL_NAME}
BEAT_PACKAGE_NAME?=${BEAT_NAME}
BEAT_INDEX_PREFIX?=${BEAT_NAME}
BEAT_URL?=https://www.elastic.co/beats/${BEAT_NAME} ## @packaging Link to the homepage of the application
BEAT_DOC_URL?=https://www.elastic.co/guide/en/beats/${BEAT_NAME}/current/index.html ## @packaging Link to the user documentation of the application
BEAT_LICENSE?=ASL 2.0 ## @packaging Software license of the application
BEAT_VENDOR?=Elastic ## @packaging Name of the vendor of the application
BEAT_GOPATH=$(firstword $(subst :, ,${GOPATH}))
BEAT_REF_YAML?=true
ES_BEATS?=..## @community_beat Must be set to ./vendor/github.com/elastic/beats. It must always be a relative path.
ifeq ($(shell uname -s), AIX)
	# Don't list packages with docker or kubernetes on AIX
	GOPACKAGES?=$(shell go list ${BEAT_IMPORT_PATH}/... | grep -v /vendor/ | grep -v /scripts/cmd/ | grep -v docker | grep -v kubernetes | grep -v /libbeat/tests/compose )
else
	GOPACKAGES?=$(shell go list ${BEAT_IMPORT_PATH}/... | grep -v /vendor/ | grep -v /scripts/cmd/ )
endif
PACKER_TEMPLATES_DIR?=${ES_BEATS}/dev-tools/packer ## @Building Directory of templates that are used by "make release"
NOTICE_FILE?=../NOTICE.txt
LICENSE_FILE?=../licenses/APACHE-LICENSE-2.0.txt
ELASTIC_LICENSE_FILE?=../licenses/ELASTIC-LICENSE.txt
SECCOMP_BINARY?=${BEAT_NAME}
SECCOMP_BLACKLIST?=${ES_BEATS}/libbeat/common/seccomp/seccomp-profiler-blacklist.txt
SECCOMP_ALLOWLIST?=${ES_BEATS}/libbeat/common/seccomp/seccomp-profiler-allow.txt
INSTALL_CMD?=install
export INSTALL_FLAG
export INSTALL_CMD
MAGE_PRESENT := $(shell command -v mage 2> /dev/null)
MAGE_IMPORT_PATH?=github.com/magefile/mage
export MAGE_IMPORT_PATH

space:=$() #
comma:=,

ARCH?=$(shell uname -m)
# Hidden directory to install dependencies for jenkins
export PATH := ./bin:$(PATH)
GOFILES = $(shell find . -type f -name '*.go' 2>/dev/null)
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "*/vendor/*" 2>/dev/null)
GOFILES_ALL = $(GOFILES) $(shell find $(ES_BEATS) -type f -name '*.go' 2>/dev/null)
GOPACKAGES_STRESSTESTS=$(shell find . -name '*.go' 2>/dev/null | xargs grep -l '\+build.*stresstest' | xargs -n1 dirname | uniq)
SHELL=bash
ES_HOST?=elasticsearch
ES_PORT?=9200
ES_USER?=beats
ES_PASS?=testing
ES_HOST_SSL?=elasticsearchssl
ES_PORT_SSL?=9200
KIBANA_HOST?=kibana
KIBANA_PORT?=5601
# Kibana's Elaticsearch user
KIBANA_ES_USER?=beats
KIBANA_ES_PASS?=testing
# The beat's Kibana user
BEAT_KIBANA_USER?=${KIBANA_ES_USER}
BEAT_KIBANA_PASS?=${KIBANA_ES_PASS}
# Define a superuser, e.g. for initial setup
ES_SUPERUSER_USER?=admin
ES_SUPERUSER_PASS?=changeme
PWD=$(shell pwd)
BUILD_DIR?=$(shell pwd)/build
PKG_BUILD_DIR?=$(BUILD_DIR)/package${PKG_SUFFIX}
PKG_UPLOAD_DIR?=$(BUILD_DIR)/upload
COVERAGE_DIR?=${BUILD_DIR}/coverage
COVERAGE_TOOL?=${BEAT_GOPATH}/bin/gotestcover
COVERAGE_TOOL_REPO?=github.com/pierrre/gotestcover
TESTIFY_TOOL_REPO?=github.com/stretchr/testify/assert
NOW=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GOBUILD_FLAGS?=-ldflags "-X github.com/elastic/beats/libbeat/version.buildTime=$(NOW) -X github.com/elastic/beats/libbeat/version.commit=$(COMMIT_ID)"
GOIMPORTS=goimports
GOIMPORTS_REPO?=golang.org/x/tools/cmd/goimports
GOIMPORTS_LOCAL_PREFIX?=github.com/elastic
GOLINT=golint
GOLINT_REPO?=golang.org/x/lint/golint
PROCESSES?= 4
TIMEOUT?= 90
PYTHON_TEST_FILES?=$(shell find . -type f -name 'test_*.py' -not -path "*/build/*" -not -path "*/vendor/*" 2>/dev/null)
PYTEST_OPTIONS?=--timeout=$(TIMEOUT) --durations=20 --junit-xml=${BUILD_DIR}/TEST-system.xml ## @testing the options to pass when calling pytest
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
DOCKER_PULL?=1 ## @miscellaneous If set to 0, disable the force pull on docker-compose build
GOPACKAGES_COMMA_SEP=$(subst $(space),$(comma),$(strip ${GOPACKAGES}))
PYTHON_EXE?=python3
PYTHON_ENV?=${BUILD_DIR}/python-env
PYTHON_ENV_EXE=${PYTHON_ENV}/bin/$(notdir ${PYTHON_EXE})
PIP_INSTALL_PARAMS?=
BUILDID?=$(shell git rev-parse HEAD) ## @Building The build ID
VENV_PARAMS?=
INTEGRATION_TESTS?=
FIND?=. ${PYTHON_ENV}/bin/activate; find . -type f -not -path "*/vendor/*" -not -path "*/build/*" -not -path "*/.git/*"
PERM_EXEC?=$(shell [ `uname -s` = "Darwin" ] && echo "+111" || echo "/a+x")
XPACK_ONLY?=false
XPACK_DIR=${ES_BEATS}/x-pack/${BEAT_NAME}

ifeq ($(DOCKER_CACHE),0)
	DOCKER_NOCACHE=--no-cache
endif

ifeq ($(DOCKER_PULL),1)
	DOCKER_FORCE_PULL=--pull
endif

# Conditionally enable the race detector when RACE_DETECTOR=1.
ifeq ($(RACE_DETECTOR),1)
	RACE=-race
endif

#
# Includes
#
include $(ES_BEATS)/dev-tools/make/mage-install.mk

### BUILDING ###

.DEFAULT_GOAL := ${BEAT_NAME}

${BEAT_NAME}: $(GOFILES_ALL) ## @build build the beat application
	go build $(GOBUILD_FLAGS)

# Create test coverage binary
${BEAT_NAME}.test: $(GOFILES_ALL)
	@go build -o /dev/null
	@go test $(RACE) -c -coverpkg ${GOPACKAGES_COMMA_SEP}

.PHONY: crosscompile
crosscompile: ## @build Cross-compile beat for the OS'es specified in GOX_OS variable. The binaries are placed in the build/bin directory.
crosscompile: $(GOFILES)
	go ${INSTALL_CMD} github.com/mitchellh/gox
	mkdir -p ${BUILD_DIR}/bin
	gox -output="${BUILD_DIR}/bin/{{.Dir}}-{{.OS}}-{{.Arch}}" -os="$(strip $(GOX_OS))" -osarch="$(strip $(GOX_OSARCH))" ${GOX_FLAGS}

.PHONY: check
check: check-headers mage ## @build Checks project and source code if everything is according to standard
	@mage check

.PHONY: $(.OVER)check-headers
$(.OVER)check-headers: mage
ifndef CHECK_HEADERS_DISABLED
	@mage checkLicenseHeaders
endif

.PHONY: $(.OVER)add-headers
$(.OVER)add-headers: mage
ifndef CHECK_HEADERS_DISABLED
	@mage addLicenseHeaders
endif

.PHONY: fmt
fmt: add-headers python-env ## @build Runs `goimports -l -w` and `autopep8`on the project's source code, modifying any files that do not match its style.
	@go ${INSTALL_CMD} $(GOIMPORTS_REPO)
	@goimports -local ${GOIMPORTS_LOCAL_PREFIX} -l -w ${GOFILES_NOVENDOR}
	@${FIND} -name '*.py' -exec ${PYTHON_ENV}/bin/autopep8 --in-place --max-line-length 120  {} \;

.PHONY: clean
clean:: mage ## @build  Cleans up all files generated by the build steps
	@mage clean
	@mage -clean

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
	go ${INSTALL_CMD} ${COVERAGE_TOOL_REPO}
	# testify is needed for unit and integration tests
	go ${INSTALL_CMD} ${TESTIFY_TOOL_REPO}

.PHONY: unit-tests
unit-tests: ## @testing Runs the unit tests with coverage.  Race is not enabled for unit tests because tests run much slower.
unit-tests: prepare-tests
	GOFLAGS="${INSTALL_FLAG}" \
		$(COVERAGE_TOOL) $(RACE) -coverprofile=${COVERAGE_DIR}/unit.cov  ${GOPACKAGES}

.PHONY: unit
unit: ## @testing Runs the unit tests without coverage reports.
	go test $(RACE) ${GOPACKAGES}

.PHONY: integration-tests
integration-tests: ## @testing Run integration tests. Unit tests are run as part of the integration tests.
integration-tests: prepare-tests mage
	rm -f docker-compose.yml.lock
	GOFLAGS="${INSTALL_FLAG}" \
		$(COVERAGE_TOOL) -tags=integration $(RACE) -coverprofile=${COVERAGE_DIR}/integration.cov ${GOPACKAGES}

.PHONY: integration-tests-environment
integration-tests-environment:  ## @testing Runs the integration inside a virtual environment. This can be run on any docker-machine (local, remote)
integration-tests-environment: prepare-tests build-image
	# If you want to get logs on CI when the `docker-compose` fails, you can
	# add  `|| ${DOCKER_COMPOSE} logs --tail 200` after the command.
	#
	# This will make docker-compose command to display the logs on stdout on error, It's not enabled
	# by default because it can create noise if the test inside the container fails.
	${DOCKER_COMPOSE} run \
	  -e RACE_DETECTOR=$(RACE_DETECTOR) \
	  -e DOCKER_COMPOSE_PROJECT_NAME=${DOCKER_COMPOSE_PROJECT_NAME} \
	  -e TEST_ENVIRONMENT=${TEST_ENVIRONMENT} \
	  -e BEATS_INSIDE_INTEGRATION_TEST_ENV=${BEATS_INSIDE_INTEGRATION_TEST_ENV} \
	  -e GOFLAGS=${INSTALL_FLAG} \
	  beat make integration-tests

# Runs the system tests
.PHONY: system-tests
system-tests: ## @testing Runs the system tests
system-tests: prepare-tests ${BEAT_NAME}.test python-env
	. ${PYTHON_ENV}/bin/activate; INTEGRATION_TESTS=${INTEGRATION_TESTS} TESTING_ENVIRONMENT=${TESTING_ENVIRONMENT} DOCKER_COMPOSE_PROJECT_NAME=${DOCKER_COMPOSE_PROJECT_NAME} pytest ${PYTHON_TEST_FILES} ${PYTEST_OPTIONS}
	${PYTHON_ENV_EXE} ${ES_BEATS}/dev-tools/aggregate_coverage.py -o ${COVERAGE_DIR}/system.cov ${BUILD_DIR}/system-tests/run

# Runs the system tests
.PHONY: system-tests-environment
system-tests-environment:  ## @testing Runs the system tests inside a virtual environment. This can be run on any docker-machine (local, remote)
system-tests-environment: prepare-tests build-image
	${DOCKER_COMPOSE} run \
		-e INTEGRATION_TESTS=1 \
		-e TESTING_ENVIRONMENT=${TESTING_ENVIRONMENT} \
		-e DOCKER_COMPOSE_PROJECT_NAME=${DOCKER_COMPOSE_PROJECT_NAME} \
		-e PYTHON_EXE=${PYTHON_EXE} \
		-e GOFLAGS=${INSTALL_FLAG} \
		beat make system-tests

.PHONY: fast-system-tests
fast-system-tests: ## @testing Runs system tests without coverage reports and in parallel
fast-system-tests: ${BEAT_NAME}.test python-env
	. ${PYTHON_ENV}/bin/activate; pytest ${PYTHON_TEST_FILES} ${PYTEST_OPTIONS}

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
	. ${PYTHON_ENV}/bin/activate; LOAD_TESTS=1 pytest ${PYTHON_TEST_FILES} --processes=$(PROCESSES) --timeout=$(TIMEOUT) -a 'load'

# Sets up the virtual python environment
.PHONY: python-env
python-env: export CRYPTOGRAPHY_DONT_BUILD_RUST=1 
python-env: ${ES_BEATS}/libbeat/tests/system/requirements.txt
	@test -e ${PYTHON_ENV}/bin/activate || ${PYTHON_EXE} -m venv ${VENV_PARAMS} ${PYTHON_ENV}
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
coverage-report: python-env
	${PYTHON_ENV_EXE} ${ES_BEATS}/dev-tools/aggregate_coverage.py -o ${COVERAGE_DIR}/full.cov ${COVERAGE_DIR}
	go tool cover -html=${COVERAGE_DIR}/full.cov -o ${COVERAGE_DIR}/full.html
	test ! -s ${COVERAGE_DIR}/integration.cov   || go tool cover -html=${COVERAGE_DIR}/integration.cov   -o ${COVERAGE_DIR}/integration.html
	test ! -s ${COVERAGE_DIR}/system.cov || go tool cover -html=${COVERAGE_DIR}/system.cov -o ${COVERAGE_DIR}/system.html
	test ! -s ${COVERAGE_DIR}/unit.cov   || go tool cover -html=${COVERAGE_DIR}/unit.cov   -o ${COVERAGE_DIR}/unit.html


.PHONY: fields
fields: mage
	@mage fields

.PHONY: config
config: mage
	@mage config

ifndef EXCLUDE_COMMON_UPDATE_TARGET
.PHONY: update
update: python-env fields collect config ## @build Update expects the most recent version of libbeat in the GOPATH
	@echo "Updating generated files for ${BEAT_NAME}"

ifneq ($(shell [[ $(BEAT_NAME) == libbeat || $(BEAT_NAME) == metricbeat ]] && echo true ),true)
	mkdir -p include
	go run ${INSTALL_FLAG} ${ES_BEATS}/dev-tools/cmd/asset/asset.go -license $(LICENSE) -pkg include -in fields.yml -out include/fields.go $(BEAT_NAME)
endif

ifneq ($(shell [[ $(BEAT_NAME) == libbeat || $(BEAT_NAME) == metricbeat ]] && echo true ),true)
	@# Update docs
	@mkdir -p docs
	@${PYTHON_ENV_EXE} ${ES_BEATS}/libbeat/scripts/generate_fields_docs.py $(PWD)/fields.yml ${BEAT_TITLE} ${ES_BEATS}
endif

	@mkdir -p $(PWD)/_meta/kibana.generated
	@# Generate Kibana index pattern and copy dashboard files
	if [ -d $(PWD)/_meta/kibana ]; then \
		cp -pr $(PWD)/_meta/kibana/* $(PWD)/_meta/kibana.generated ; \
	fi
endif

.PHONY: docs
docs:  ## @build Builds the documentation for the beat
	@if [ -d $(XPACK_DIR) ]; then \
		sh ${ES_BEATS}/script/build_docs.sh ${BEAT_NAME} ${BEAT_PATH}/docs ${BUILD_DIR} ${XPACK_DIR}; \
	else \
		sh ${ES_BEATS}/script/build_docs.sh ${BEAT_NAME} ${BEAT_PATH}/docs ${BUILD_DIR}; \
	fi

.PHONY: docs-preview
docs-preview:  ## @build Previews the documentation for the beat in the browser
	PREVIEW=1 $(MAKE) docs

### SETUP commands ###
ES_URL?=http://localhost:9200
KIBANA_URL?=http://localhost:5601

.PHONY: $(.OVER)import-dashboards
$(.OVER)import-dashboards: update ${BEAT_NAME}
	${BEAT_GOPATH}/src/${BEAT_PATH}/${BEAT_NAME} setup -E setup.dashboards.directory=${PWD}/_meta/kibana.generated -E setup.kibana.host=${KIBANA_URL} --dashboards

### CONTAINER ENVIRONMENT ####

# Builds the environment to test beat
.PHONY: build-image
build-image: write-environment
	${DOCKER_COMPOSE} build ${DOCKER_NOCACHE} ${DOCKER_FORCE_PULL} --force-rm

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
	# set ENV variables for beat
	echo "ES_HOST=${ES_HOST}" >> ${BUILD_DIR}/test.env
	echo "ES_PORT=${ES_PORT}" >> ${BUILD_DIR}/test.env
	echo "ES_HOST_SSL=${ES_HOST_SSL}" >> ${BUILD_DIR}/test.env
	echo "ES_PORT_SSL=${ES_PORT_SSL}" >> ${BUILD_DIR}/test.env
	echo "ES_USER=${ES_USER}" >> ${BUILD_DIR}/test.env
	echo "ES_PASS=${ES_PASS}" >> ${BUILD_DIR}/test.env
	echo "ES_SUPERUSER_USER=${ES_SUPERUSER_USER}" >> ${BUILD_DIR}/test.env
	echo "ES_SUPERUSER_PASS=${ES_SUPERUSER_PASS}" >> ${BUILD_DIR}/test.env
	echo "KIBANA_HOST=${KIBANA_HOST}" >> ${BUILD_DIR}/test.env
	echo "KIBANA_PORT=${KIBANA_PORT}" >> ${BUILD_DIR}/test.env
	echo "KIBANA_USER=${BEAT_KIBANA_USER}" >> ${BUILD_DIR}/test.env
	echo "KIBANA_PASS=${BEAT_KIBANA_PASS}" >> ${BUILD_DIR}/test.env
	# set Kibana ENV variables
	echo "ELASTICSEARCH_URL=${ES_HOST}:${ES_PORT}" > ${BUILD_DIR}/kibana.test.env
	echo "ELASTICSEARCH_USERNAME=${KIBANA_ES_USER}" >> ${BUILD_DIR}/kibana.test.env
	echo "ELASTICSEARCH_PASSWORD=${KIBANA_ES_PASS}" >> ${BUILD_DIR}/kibana.test.env

# Tails the environment logs
.PHONY: env-logs
env-logs:
	${DOCKER_COMPOSE} logs -f

fix-permissions:
	sh ${ES_BEATS}/script/fix_permissions.sh "${PWD}"

set_version: ## @packaging VERSION=x.y.z set the version of the beat to x.y.z
	${ES_BEATS}/dev-tools/set_version ${VERSION}

get_version: ## @packaging get the version of the beat.
	@${ES_BEATS}/dev-tools/get_version

help: ## @help Show this help.
	@${PYTHON_ENV_EXE} ${ES_BEATS}/libbeat/scripts/generate_makefile_doc.py $(MAKEFILE_LIST)

help_variables: ## @help Show Makefile customizable variables.
	@${PYTHON_ENV_EXE} ${ES_BEATS}/libbeat/scripts/generate_makefile_doc.py --variables $(MAKEFILE_LIST)

# Generates a seccomp whitelist policy for the binary pointed to by
# SECCOMP_BINARY.
.PHONY: seccomp
seccomp:
	@go ${INSTALL_CMD} github.com/elastic/go-seccomp-bpf/cmd/seccomp-profiler
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

.PHONY: release
release: mage
	@mage package

.PHONY: snapshot
snapshot: mage
	@SNAPSHOT=true mage package

ifdef NO_COLLECT
.PHONY: collect
collect:
endif
