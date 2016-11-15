
### VARIABLE SETUP ###

BEATNAME?=libbeat
BEAT_DESCRIPTION?=Sends events to Elasticsearch or Logstash
BEAT_DIR?=github.com/elastic/beats/${BEATNAME}
ES_BEATS?=..
GOPACKAGES?=${BEAT_DIR}/...

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
GOFILES = $(shell find . -type f -name '*.go')
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "*/vendor/*")
GOFILES_ALL = $(GOFILES) $(shell find $(ES_BEATS) -type f -name '*.go')
SHELL=bash
ES_HOST?="elasticsearch"
PWD=$(shell pwd)
BUILD_DIR?=$(shell pwd)/build
COVERAGE_DIR=${BUILD_DIR}/coverage
COVERAGE_TOOL=${GOPATH}/bin/gotestcover
COVERAGE_TOOL_REPO=github.com/elastic/beats/vendor/github.com/pierrre/gotestcover
PROCESSES?= 4
TIMEOUT?= 90
TEST_ENVIRONMENT?=false
SYSTEM_TESTS?=false
GOX_OS?=linux darwin windows solaris freebsd netbsd openbsd
TESTING_ENVIRONMENT?=snapshot
DOCKER_COMPOSE?=docker-compose -f ${PWD}/../testing/environments/base.yml -f ${PWD}/../testing/environments/${TESTING_ENVIRONMENT}.yml -f docker-compose.yml
DOCKER_CACHE?=1 # If set to 0, all docker images are created without cache
GOPACKAGES_COMMA_SEP=$(subst $(space),$(comma),$(strip ${GOPACKAGES}))
PYTHON_ENV?=${BUILD_DIR}/python-env
BUILDID?=$(shell git rev-parse HEAD)

CGO?=false

# Cross compiling targets
TARGETS?="linux/amd64 linux/386 windows/amd64 windows/386 darwin/amd64"
# Cross compiling targets to be build on debian 6. This only applies if CGO is enabled
TARGETS_OLD?=""
PACKAGES?=${BEATNAME}/deb ${BEATNAME}/rpm ${BEATNAME}/darwin ${BEATNAME}/win ${BEATNAME}/bin
SNAPSHOT?=yes

ifeq ($(DOCKER_CACHE),0)
	DOCKER_NOCACHE=--no-cache
endif

# Conditionally enable the race detector when RACE_DETECTOR=1.
ifeq ($(RACE_DETECTOR),1)
	RACE=-race
endif


### BUILDING ###

# Builds beat
${BEATNAME}: $(GOFILES_ALL)
	go build

# Create test coverage binary
${BEATNAME}.test: $(GOFILES_ALL)
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
	rm -rf build ${BEATNAME} ${BEATNAME}.test ${BEATNAME}.exe ${BEATNAME}.test.exe _meta/fields.generated.yml

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
	go get ${COVERAGE_TOOL_REPO}

# Runs the unit tests with coverage
# Race is not enabled for unit tests because tests run much slower.
.PHONY: unit-tests
unit-tests: prepare-tests
	$(COVERAGE_TOOL) $(RACE) -coverprofile=${COVERAGE_DIR}/unit.cov  ${GOPACKAGES}

# Runs the unit tests without coverage reports.
.PHONY: unit
unit:
	go test $(RACE) ${GOPACKAGES}

# Run integration tests. Unit tests are run as part of the integration tests.
.PHONY: integration-tests
integration-tests: prepare-tests
	$(COVERAGE_TOOL) -tags=integration $(RACE) -coverprofile=${COVERAGE_DIR}/integration.cov  ${GOPACKAGES}

# Runs the integration inside a virtual environment. This can be run on any docker-machine (local, remote)
.PHONY: integration-tests-environment
integration-tests-environment: prepare-tests build-image
	${DOCKER_COMPOSE} run beat make integration-tests RACE_DETECTOR=$(RACE_DETECTOR)

# Runs the system tests
.PHONY: system-tests
system-tests: ${BEATNAME}.test prepare-tests python-env
	. ${PYTHON_ENV}/bin/activate; INTEGRATION_TESTS=1 nosetests -w tests/system --process-timeout=$(TIMEOUT) --with-timer
	python ${ES_BEATS}/dev-tools/aggregate_coverage.py -o ${COVERAGE_DIR}/system.cov ./build/system-tests/run

# Runs the system tests
.PHONY: system-tests-environment
system-tests-environment: prepare-tests build-image
	${DOCKER_COMPOSE} run beat make system-tests

# Runs system tests without coverage reports and in parallel
.PHONY: fast-system-tests
fast-system-tests: ${BEATNAME}.test python-env
	. ${PYTHON_ENV}/bin/activate; nosetests -w tests/system --processes=$(PROCESSES) --process-timeout=$(TIMEOUT)

# Run benchmark tests
.PHONY: benchmark-tests
benchmark-tests:
	# No benchmark tests exist so far
	#go test -bench=. ${GOPACKAGES}

# Run load tests
.PHONY: load-tests
load-tests:
	. ${PYTHON_ENV}/bin/activate; LOAD_TESTS=1  nosetests -w tests/system --processes=$(PROCESSES) --process-timeout=$(TIMEOUT) -a 'load'

# Sets up the virtual python environment
.PHONY: python-env
python-env: ${ES_BEATS}/libbeat/tests/system/requirements.txt
	@test -d ${PYTHON_ENV} || virtualenv ${PYTHON_ENV}
	@. ${PYTHON_ENV}/bin/activate && pip install -q --upgrade pip ; \
	if [ -a ./tests/system/requirements.txt ] && [ ! ${ES_BEATS}/libbeat/tests/system/requirements.txt -ef ./tests/system/requirements.txt ] ; then \
		. ${PYTHON_ENV}/bin/activate && pip install -qUr ${ES_BEATS}/libbeat/tests/system/requirements.txt -Ur ./tests/system/requirements.txt ; \
	else \
		. ${PYTHON_ENV}/bin/activate && pip install -qUr ${ES_BEATS}/libbeat/tests/system/requirements.txt ; \
	fi

# Runs unit and system tests without coverage reports
.PHONY: test
test: unit
	if [ $(SYSTEM_TESTS) = true ]; then \
		 $(MAKE) fast-system-tests; \
	fi

# Runs all tests and generates the coverage reports
.PHONY: testsuite
testsuite: clean collect
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
			$(MAKE) system-tests-environment; \
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
update: python-env collect

	# Update config
	echo "Update config file"
	-rm -f ${BEATNAME}.yml
	cat _meta/beat.yml ${ES_BEATS}/libbeat/_meta/config.yml | sed -e "s/beatname/${BEATNAME}/g" > ${BEATNAME}.yml
	-rm -f ${BEATNAME}.full.yml
	cat _meta/beat.yml ${ES_BEATS}/libbeat/_meta/config.full.yml | sed -e "s/beatname/${BEATNAME}/g" > ${BEATNAME}.full.yml

	# Check if also a full config exist (optional)
	if [ -a _meta/beat.full.yml ] ; \
    then \
		cat _meta/beat.full.yml ${ES_BEATS}/libbeat/_meta/config.full.yml | sed -e "s/beatname/${BEATNAME}/g" > ${BEATNAME}.full.yml ; \
	fi;

	# make sure generated docs in libbeat are up-to-date
	$(MAKE) -C ${ES_BEATS}/libbeat fields

	# Update docs
	. ${PYTHON_ENV}/bin/activate && python ${ES_BEATS}/libbeat/scripts/generate_fields_docs.py $(PWD) ${BEATNAME} ${ES_BEATS}

	# Generate index templates
	. ${PYTHON_ENV}/bin/activate && python ${ES_BEATS}/libbeat/scripts/generate_template.py $(PWD) ${BEATNAME} ${ES_BEATS}
	. ${PYTHON_ENV}/bin/activate && python ${ES_BEATS}/libbeat/scripts/generate_template.py --es2x $(PWD) ${BEATNAME} ${ES_BEATS}

	# Update docs version
	cp ${ES_BEATS}/libbeat/docs/version.asciidoc docs/version.asciidoc

	# Generate index-pattern
	echo "Generate index pattern"
	-rm -f $(PWD)/_meta/kibana/index-pattern/${BEATNAME}.json
	mkdir -p $(PWD)/_meta/kibana/index-pattern
	. ${PYTHON_ENV}/bin/activate && python ${ES_BEATS}/libbeat/scripts/generate_index_pattern.py --index '${BEATNAME}-*' --libbeat ${ES_BEATS}/libbeat --beat $(PWD)


# Builds the documents for the beat
.PHONY: docs
docs:
	sh ${ES_BEATS}/libbeat/scripts/build_docs.sh ${BEATNAME}

# Preview the documents for the beat in the browser
.PHONY: docs-preview
docs-preview:
	if [ ! -d "build/docs" ]; then $(MAKE) docs; fi;
	${BUILD_DIR}/docs/build_docs.pl --chunk=1 -open chunk=1 -open --doc ${GOPATH}/src/github.com/elastic/beats/${BEATNAME}/docs/index.asciidoc -out ${BUILD_DIR}/html_docs


### KIBANA FILES HANDLING ###
ES_URL?=http://localhost:9200

.PHONY: export-dashboards
export-dashboards: python-env update
	. ${PYTHON_ENV}/bin/activate && python ${ES_BEATS}/dev-tools/export_dashboards.py --url ${ES_URL} --dir $(shell pwd)/_meta/kibana --regex ${BEATNAME}-*

.PHONY: import-dashboards
import-dashboards: update
	$(MAKE) -C ${ES_BEATS}/libbeat/dashboards import_dasboards
	${ES_BEATS}/libbeat/dashboards/import_dashboards -es ${ES_URL} -dir ${PWD}/_meta/kibana

### CONTAINER ENVIRONMENT ####

# Builds the environment to test beat
.PHONY: build-image
build-image: write-environment
	${DOCKER_COMPOSE} build ${DOCKER_NOCACHE}

# Runs the environment so the redis and elasticsearch can also be used for local development
# To use it for running the test, set ES_HOST and REDIS_HOST environment variable to the ip of your docker-machine.
.PHONY: start-environment
start-environment: stop-environment
	${DOCKER_COMPOSE} up -d

.PHONY: stop-environment
stop-environment:
	-${DOCKER_COMPOSE} stop
	-${DOCKER_COMPOSE} rm -f -v -a

.PHONY: write-environment
write-environment:
	mkdir -p ${BUILD_DIR}
	echo "ES_HOST=${ES_HOST}" > ${BUILD_DIR}/test.env
	echo "ES_PORT=9200" >> ${BUILD_DIR}/test.env
	echo "ES_USER=beats" >> ${BUILD_DIR}/test.env
	echo "ES_PASS=testing" >> ${BUILD_DIR}/test.env

# Tails the environment logs
.PHONY: env-logs
env-logs:
	${DOCKER_COMPOSE} logs -f


### Packaging targets ####

# Installs the files that need to get to the home path on installations
HOME_PREFIX?=/tmp/${BEATNAME}
.PHONY: install-home
install-home:
	install -d -m 755 ${HOME_PREFIX}/scripts/
	install -m 755 ${ES_BEATS}/libbeat/scripts/migrate_beat_config_1_x_to_5_0.py ${HOME_PREFIX}/scripts/

# Prepares for packaging. Builds binaries and creates homedir data
.PHONY: prepare-package
prepare-package:
	# cross compile on ubuntu
	docker run --rm \
		-v $(abspath ${ES_BEATS}/dev-tools/packer/xgo-scripts):/scripts \
		-v $(abspath ../):/source \
		-v $(BUILD_DIR):/build \
		-e PUREGO="yes" \
		-e PACK=${BEATNAME} \
		-e BEFORE_BUILD=before_build.sh \
		-e SOURCE=/source \
		-e TARGETS=${TARGETS} \
		-e BUILDID=${BUILDID} \
		tudorg/beats-builder \
		${BEAT_DIR}

# Prepares for packaging. Builds binaries with cgo
.PHONY: prepare-package-cgo
prepare-package-cgo:

	# cross compile on ubuntu
	docker run --rm \
		-v $(abspath ${ES_BEATS}/dev-tools/packer/xgo-scripts):/scripts \
		-v $(abspath ../):/source \
		-v $(BUILD_DIR):/build \
		-e PACK=${BEATNAME} \
		-e BEFORE_BUILD=before_build.sh \
		-e SOURCE=/source \
		-e TARGETS=${TARGETS} \
		-e BUILDID=${BUILDID} \
		tudorg/beats-builder \
		${BEAT_DIR}

	# linux builds on debian 6 for compatibility
	docker run --rm \
		-v ${BUILD_DIR}:/build \
		-v $(abspath ${ES_BEATS}/dev-tools/packer/xgo-scripts):/scripts \
		-v $(abspath ..):/source \
		-e PACK=${BEATNAME} \
		-e BEFORE_BUILD=before_build.sh \
		-e SOURCE=/source \
		-e TARGETS=${TARGETS_OLD} \
		-e BUILDID=${BUILDID} \
		tudorg/beats-builder-deb6 \
		${BEAT_DIR}

# Prepares images for packaging
.PHONY: package-setup
package-setup:
	$(MAKE) -C ${ES_BEATS}/dev-tools/packer deps images

# Create binary packages for the beat.
.PHONY: package
package: package-setup

	echo "Start building packages for ${BEATNAME}"

	mkdir -p ${BUILD_DIR}/upload

	# Generates the package.yml file with all information needed to create packages
	echo "beat_name: ${BEATNAME}" > ${BUILD_DIR}/package.yml
	echo "beat_url: https://${BEAT_DIR}" >> ${BUILD_DIR}/package.yml
	echo "beat_repo: ${BEAT_DIR}" >> ${BUILD_DIR}/package.yml
	echo "beat_description: ${BEAT_DESCRIPTION}" >> ${BUILD_DIR}/package.yml
	cat ${ES_BEATS}/dev-tools/packer/version.yml >> ${BUILD_DIR}/package.yml

	if [ $(CGO) = true ]; then \
		 $(MAKE) prepare-package-cgo; \
	else \
		$(MAKE) prepare-package; \
	fi

	SNAPSHOT=${SNAPSHOT} BUILDID=${BUILDID} BEAT_DIR=${BEAT_DIR} BUILD_DIR=${BUILD_DIR} $(MAKE) -C ${ES_BEATS}/dev-tools/packer ${PACKAGES} ${BUILD_DIR}/upload/build_id.txt

	echo "Finished packages for ${BEATNAME}"

package-dashboards: package-setup
	mkdir -p ${BUILD_DIR}
	cp -r _meta/kibana ${BUILD_DIR}/dashboards
	# build the dashboards package
	BEATNAME=${BEATNAME} BUILD_DIR=${BUILD_DIR} SNAPSHOT=$(SNAPSHOT) $(MAKE) -C ${ES_BEATS}/dev-tools/packer package-dashboards ${shell pwd}/build/upload/build_id.txt
