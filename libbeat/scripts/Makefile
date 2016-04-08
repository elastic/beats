
### VARIABLE SETUP ###

BEATNAME?=libbeat
BEAT_DIR?=github.com/elastic/beats
ES_BEATS?=..
GOPACKAGES?=${BEAT_DIR}/${BEATNAME}/...

# Makefile for a custom beat that includes this libbeat/scripts/Makefile:
# if glide is used to manage vendor dependencies,
#     BEATNAME=mybeat
#     BEAT_DIR=github.com/mybeat
#     ES_BEATS=./vendor/github.com/elastic/beats
#     GOPACKAGES=$(shell glide novendor)
#     include $(ES_BEATS)/libbeat/scripts/Makefile
# else
#     BEATNAME=mybeat
#     BEAT_DIR=github.com/mybeat
#     ES_BEATS=$(GOPATH)/src/github.com/elastic/beats
#     GOPACKAGES=$(shell go list ${BEAT_DIR}/... | grep -v /vendor/)
#     include $(ES_BEATS)/libbeat/scripts/Makefile

space:=$() #
comma:=,

ARCH?=$(shell uname -m)
# Hidden directory to install dependencies for jenkins
export PATH := ./bin:$(PATH)
export GO15VENDOREXPERIMENT=1
GOFILES = $(shell find . -type f -name '*.go')
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "*/vendor/*")
SHELL=bash
ES_HOST?="elasticsearch"
BUILD_DIR?=$(shell pwd)/build
COVERAGE_DIR=${BUILD_DIR}/coverage
PROCESSES?= 4
TIMEOUT?= 90
TEST_ENVIRONMENT?=false
SYSTEM_TESTS?=false
GOX_OS?=linux darwin windows solaris freebsd netbsd openbsd
DOCKER_COMPOSE?=docker-compose -f docker-compose.yml
GOPACKAGES_COMMA_SEP=$(subst $(space),$(comma),$(strip ${GOPACKAGES}))
PYTHON_ENV?=${BUILD_DIR}/python-env

# Conditionally enable the race detector when RACE_DETECTOR=1.
ifeq ($(RACE_DETECTOR),1)
	RACE=-race
endif


### BUILDING ###

# Builds beat
.PHONY: build
build: $(GOFILES)
	go build

# Create test coverage binary
.PHONY: buildbeat.test
buildbeat.test: $(GOFILES)
	go test $(RACE) -c -coverpkg ${GOPACKAGES_COMMA_SEP}

# Cross-compile beat for the OS'es specified in GOX_OS variable.
# The binaries are placed in the build/bin directory.
.PHONY: crosscompile
crosscompile: $(GOFILES)
	go get github.com/mitchellh/gox
	mkdir -p ${BUILD_DIR}/bin
	gox -output="${BUILD_DIR}/bin/{{.Dir}}-{{.OS}}-{{.Arch}}" -os="${GOX_OS}" ${GOX_FLAGS}

# Checks project and source code if everything is according to standard
.PHONY: check
check:
	@gofmt -l ${GOFILES_NOVENDOR} | (! grep . -q) || (echo "Code differs from gofmt's style" && false)
	go vet ${GOPACKAGES}

# Runs gofmt -w on the project's source code, modifying any files that do not
# match its style.
.PHONY: fmt
fmt:
	gofmt -l -w ${GOFILES_NOVENDOR}

# Runs gofmt -s -w on the project's source code, modifying any files that do not
# match its style.
.PHONY: simplify
simplify:
	gofmt -l -s -w ${GOFILES_NOVENDOR}

# Cleans up directory and source code with gofmt
.PHONY: clean
clean:
	rm -rf build ${BEATNAME} ${BEATNAME}.test ${BEATNAME}.exe ${BEATNAME}.test.exe

# Shortcut for continuous integration
# This should always run before merging.
.PHONY: ci
ci:
	$(MAKE)
	$(MAKE) check
	$(MAKE) testsuite

### Testing ###
# Unless stated otherwise, all tests are always run with coverage reporting enabled.


# Prepration for tests
.PHONY: prepare-tests
prepare-tests:
	mkdir -p ${COVERAGE_DIR}
	# gotestcover is needed to fetch coverage for multiple packages
	go get github.com/pierrre/gotestcover

# Runs the unit tests with coverage
# Race is not enabled for unit tests because tests run much slower.
.PHONY: unit-tests
unit-tests: prepare-tests
	$(GOPATH)/bin/gotestcover $(RACE) -coverprofile=${COVERAGE_DIR}/unit.cov  ${GOPACKAGES}


# Runs the unit tests without coverage reports.
.PHONY: unit
unit:
	go test $(RACE) ${GOPACKAGES}

# Run integration tests. Unit tests are run as part of the integration tests.
.PHONY: integration-tests
integration-tests: prepare-tests
	$(GOPATH)/bin/gotestcover -tags=integration $(RACE) -coverprofile=${COVERAGE_DIR}/integration.cov  ${GOPACKAGES}

# Runs the integration inside a virtual environment. This can be run on any docker-machine (local, remote)
.PHONY: integration-tests-environment
integration-tests-environment:
	$(MAKE) prepare-tests
	$(MAKE) build-image
	${DOCKER_COMPOSE} run beat make integration-tests RACE_DETECTOR=$(RACE_DETECTOR)

# Runs the system tests
.PHONY: system-tests
system-tests: buildbeat.test prepare-tests python-env
	. ${PYTHON_ENV}/bin/activate; nosetests -w tests/system --process-timeout=$(TIMEOUT) --with-timer
	python ${ES_BEATS}/dev-tools/aggregate_coverage.py -o ${COVERAGE_DIR}/system.cov ./build/system-tests/run

# Runs system tests without coverage reports and in parallel
.PHONY: fast-system-tests
fast-system-tests: buildbeat.test python-env
	. ${PYTHON_ENV}/bin/activate; nosetests -w tests/system --processes=$(PROCESSES) --process-timeout=$(TIMEOUT)

# Run benchmark tests
.PHONY: benchmark-tests
benchmark-tests:
	# No benchmark tests exist so far
	#go test -bench=. ${GOPACKAGES}

# Sets up the virtual python environment
.PHONY: python-env
python-env: ${ES_BEATS}/libbeat/tests/system/requirements.txt
	test -d ${PYTHON_ENV} || virtualenv ${PYTHON_ENV}
	if [ -a ./tests/system/requirements.txt ] && [ ! ${ES_BEATS}/libbeat/tests/system/requirements.txt -ef ./tests/system/requirements.txt ] ; then \
		. ${PYTHON_ENV}/bin/activate && pip install -Ur ${ES_BEATS}/libbeat/tests/system/requirements.txt -Ur ./tests/system/requirements.txt ; \
	else \
		. ${PYTHON_ENV}/bin/activate && pip install -Ur ${ES_BEATS}/libbeat/tests/system/requirements.txt ; \
	fi

# Runs unit and system tests without coverage reports
.PHONY: test
test: unit
	if [ $(SYSTEM_TESTS) = true ]; then \
		 $(MAKE) fast-system-tests; \
	fi

# Runs all tests and generates the coverage reports
.PHONY: testsuite
testsuite:
	$(MAKE) unit-tests

	# Setups environment if TEST_ENVIRONMENT is set to true
	if [ $(TEST_ENVIRONMENT) = true ]; then \
		 $(MAKE) integration-tests-environment; \
	else \
		$(MAKE) integration-tests; \
	fi

	# Runs system tests if SYSTEM_TESTS is set to true
	if [ $(SYSTEM_TESTS) = true ]; then \
		if [ $(TEST_ENVIRONMENT) = true ]; then \
			${DOCKER_COMPOSE} run beat make system-tests; \
		else \
			$(MAKE) system-tests; \
		fi \
	fi

	$(MAKE) benchmark-tests
	$(MAKE) coverage-report

# Generates a coverage report from the existing coverage files
.PHONY: coverage-report
coverage-report:
	python ${ES_BEATS}/dev-tools/aggregate_coverage.py -o ${COVERAGE_DIR}/full.cov ${COVERAGE_DIR}
	go tool cover -html=${COVERAGE_DIR}/full.cov -o ${COVERAGE_DIR}/full.html
	test ! -s ${COVERAGE_DIR}/integration.cov   || go tool cover -html=${COVERAGE_DIR}/integration.cov   -o ${COVERAGE_DIR}/integration.html
	test ! -s ${COVERAGE_DIR}/system.cov || go tool cover -html=${COVERAGE_DIR}/system.cov -o ${COVERAGE_DIR}/system.html
	test ! -s ${COVERAGE_DIR}/unit.cov   || go tool cover -html=${COVERAGE_DIR}/unit.cov   -o ${COVERAGE_DIR}/unit.html

# Update expects the most recent version of libbeat in the GOPATH
.PHONY: update
update:
	bash ${ES_BEATS}/libbeat/scripts/update.sh ${BEATNAME} ${BEAT_DIR}/${BEATNAME} ${ES_BEATS}/libbeat

### KIBANA FILES HANDLING ###
ES_URL?=http://localhost:9200/

.PHONY: update
export-dashboards:
	python ${ES_BEATS}/dev-tools//export_dashboards.py --url ${ES_URL} --dir $(shell pwd)/etc/kibana --beat ${BEATNAME}

.PHONY: update
import-dashboards:
	bash ${ES_BEATS}/dev-tools/import_dashboards.sh -url ${ES_URL} -dir $(shell pwd)/etc/kibana

### CONTAINER ENVIRONMENT ####

# Builds the environment to test beat
.PHONY: build-image
build-image: write-environment
	${DOCKER_COMPOSE} build

# Runs the environment so the redis and elasticsearch can also be used for local development
# To use it for running the test, set ES_HOST and REDIS_HOST environment variable to the ip of your docker-machine.
.PHONY: start-environment
start-environment: stop-environment
	${DOCKER_COMPOSE} up -d

.PHONY: stop-environment
stop-environment:
	-${DOCKER_COMPOSE} stop
	-${DOCKER_COMPOSE} rm -f
	-docker ps -a  | grep ${BEATNAME} | grep Exited | awk '{print $$1}' | xargs docker rm

.PHONY: write-environment
write-environment:
	mkdir -p ${BUILD_DIR}
	echo "ES_HOST=${ES_HOST}" > ${BUILD_DIR}/test.env
	echo "ES_PORT=9200" >> ${BUILD_DIR}/test.env
	echo "ES_USER=beats" >> ${BUILD_DIR}/test.env
	echo "ES_PASS=testing" >> ${BUILD_DIR}/test.env

