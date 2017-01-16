### VARIABLE SETUP ###
### Application using libbeat may override the following variables in their Makefile
BEATNAME?=libbeat ## @packaging Name of the application
BEAT_DESCRIPTION?=Sends events to Elasticsearch or Logstash ## @packaging Description of the application
BEAT_VENDOR?=Elastic ## @packaging Name of the vendor of the application
BEAT_LICENSE?=ASL 2.0 ## @packaging Software license of the application
BEAT_DOC_URL?=https://www.elastic.co/guide/en/beats/${BEATNAME}/current/index.html ## @packaging Link to the user documentation of the application
BEAT_DIR?=github.com/elastic/beats/${BEATNAME}
ES_BEATS?=..## @community_beat Must be set to ./vendor/github.com/elastic/beats
GOPACKAGES?=${BEAT_DIR}/...## @community_beat Must be set to $(shell glide novendor)
PACKER_TEMPLATES_DIR?=${ES_BEATS}/dev-tools/packer ## @Building Directory of templates that are used by "make package"
NOTICE_FILE?=../NOTICE

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
TEST_ENVIRONMENT?=false ## @testing if true, "make testsuite" runs integration tests and system tests in a dockerized test environment
SYSTEM_TESTS?=false ## @testing if true, "make test" and "make testsuite" run unit tests and system tests
GOX_OS?=linux darwin windows solaris freebsd netbsd openbsd ## @Building List of all OS to be supported by "make crosscompile".
TESTING_ENVIRONMENT?=snapshot ## @testing The name of the environment under test
DOCKER_COMPOSE_PROJECT_NAME?=${BEATNAME}_${TESTING_ENVIRONMENT} ## @testing The name of the docker-compose project used by the integration and system tests
DOCKER_COMPOSE?=TESTING_ENVIRONMENT=${TESTING_ENVIRONMENT} docker-compose -p ${DOCKER_COMPOSE_PROJECT_NAME} -f docker-compose.yml
DOCKER_CACHE?=1 ## @miscellaneous If set to 0, all docker images are created without cache
GOPACKAGES_COMMA_SEP=$(subst $(space),$(comma),$(strip ${GOPACKAGES}))
PYTHON_ENV?=${BUILD_DIR}/python-env
BUILDID?=$(shell git rev-parse HEAD) ## @Building The build ID
VIRTUALENV_PARAMS?=
INTEGRATION_TESTS?=
CGO?=false ## @building if true, Build with C Go support

# Cross compiling targets
TARGETS?="linux/amd64 linux/386 windows/amd64 windows/386 darwin/amd64" ## @building list of platforms/architecture to be built by "make package"
TARGETS_OLD?="" ## @building list of Debian6 architecture to be built by "make package" when CGO is true
PACKAGES?=${BEATNAME}/deb ${BEATNAME}/rpm ${BEATNAME}/darwin ${BEATNAME}/win ${BEATNAME}/bin ## @Building List of OS to be supported by "make package"
SNAPSHOT?=yes ## @Building If yes, builds a snapshot version
BEATS_BUILDER_IMAGE?=tudorg/beats-builder ## @Building Name of the docker image to use when packaging the application
BEATS_BUILDER_DEB6_IMAGE?=tudorg/beats-builder-deb6 ## @Building Name of the docker image to use when packaging the application for Debian 6

ifeq ($(DOCKER_CACHE),0)
	DOCKER_NOCACHE=--no-cache
endif

# Conditionally enable the race detector when RACE_DETECTOR=1.
ifeq ($(RACE_DETECTOR),1)
	RACE=-race
endif


### BUILDING ###


${BEATNAME}: $(GOFILES_ALL) ## @build build the beat application
	go build

# Create test coverage binary
${BEATNAME}.test: $(GOFILES_ALL)
	go test $(RACE) -c -coverpkg ${GOPACKAGES_COMMA_SEP}

.PHONY: crosscompile
crosscompile: ## @build Cross-compile beat for the OS'es specified in GOX_OS variable. The binaries are placed in the build/bin directory.
crosscompile: $(GOFILES)
	go get github.com/mitchellh/gox
	mkdir -p ${BUILD_DIR}/bin
	gox -output="${BUILD_DIR}/bin/{{.Dir}}-{{.OS}}-{{.Arch}}" -os="${GOX_OS}" ${GOX_FLAGS}


.PHONY: check
check:  ## @build Checks project and source code if everything is according to standard
	@gofmt -l ${GOFILES_NOVENDOR} | (! grep . -q) || (echo "Code differs from gofmt's style" && false)
	go vet ${GOPACKAGES}

.PHONY: fmt
fmt: ## @build Runs gofmt -w on the project's source code, modifying any files that do not match its style.
	gofmt -l -w ${GOFILES_NOVENDOR}

.PHONY: simplify
simplify: ## @build Runs gofmt -s -w on the project's source code, modifying any files that do not match its style.
	gofmt -l -s -w ${GOFILES_NOVENDOR}

.PHONY: clean
clean:: ## @build  Cleans up all files generated by the build steps
	rm -rf build ${BEATNAME} ${BEATNAME}.test ${BEATNAME}.exe ${BEATNAME}.test.exe _meta/fields.generated.yml

.PHONY: ci
ci:  ## @build Shortcut for continuous integration. This should always run before merging.
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

.PHONY: unit-tests
unit-tests: ## @testing Runs the unit tests with coverage.  Race is not enabled for unit tests because tests run much slower.
unit-tests: prepare-tests
	$(COVERAGE_TOOL) $(RACE) -coverprofile=${COVERAGE_DIR}/unit.cov  ${GOPACKAGES}

.PHONY: unit
unit: ## @testing Runs the unit tests without coverage reports.
	go test $(RACE) ${GOPACKAGES}

.PHONY: integration-tests
integration-tests: ## @testing Run integration tests. Unit tests are run as part of the integration tests.
integration-tests: prepare-tests
	$(COVERAGE_TOOL) -tags=integration $(RACE) -coverprofile=${COVERAGE_DIR}/integration.cov  ${GOPACKAGES}

#
.PHONY: integration-tests-environment
integration-tests-environment:  ## @testing Runs the integration inside a virtual environment. This can be run on any docker-machine (local, remote)
integration-tests-environment: prepare-tests build-image
	${DOCKER_COMPOSE} run beat make integration-tests RACE_DETECTOR=$(RACE_DETECTOR)

# Runs the system tests
.PHONY: system-tests
system-tests: ## @testing Runs the system tests
system-tests: ${BEATNAME}.test prepare-tests python-env ${ES_BEATS}/libbeat/dashboards/import_dashboards
	. ${PYTHON_ENV}/bin/activate; INTEGRATION_TESTS=${INTEGRATION_TESTS} nosetests -w tests/system --process-timeout=$(TIMEOUT) --with-timer
	python ${ES_BEATS}/dev-tools/aggregate_coverage.py -o ${COVERAGE_DIR}/system.cov ./build/system-tests/run

# Runs the system tests
.PHONY: system-tests-environment
system-tests-environment:  ## @testing Runs the system tests inside a virtual environment. This can be run on any docker-machine (local, remote)
system-tests-environment: prepare-tests build-image
	${DOCKER_COMPOSE} run -e INTEGRATION_TESTS=1 beat make system-tests


.PHONY: fast-system-tests
fast-system-tests: ## @testing Runs system tests without coverage reports and in parallel
fast-system-tests: ${BEATNAME}.test python-env
	. ${PYTHON_ENV}/bin/activate; nosetests -w tests/system --processes=$(PROCESSES) --process-timeout=$(TIMEOUT)

# Run benchmark tests
.PHONY: benchmark-tests
benchmark-tests: ## @testing Runs bechmarks (NOT YET IMPLEMENTED)
	# No benchmark tests exist so far
	#go test -bench=. ${GOPACKAGES}

# Run load tests
.PHONY: load-tests
load-tests: ## @testing Runs load tests
	. ${PYTHON_ENV}/bin/activate; LOAD_TESTS=1  nosetests -w tests/system --processes=$(PROCESSES) --process-timeout=$(TIMEOUT) -a 'load'

# Sets up the virtual python environment
.PHONY: python-env
python-env: ${ES_BEATS}/libbeat/tests/system/requirements.txt
	@test -d ${PYTHON_ENV} || virtualenv ${VIRTUALENV_PARAMS} ${PYTHON_ENV}
	@. ${PYTHON_ENV}/bin/activate && pip install -q --upgrade pip ; \
	if [ -a ./tests/system/requirements.txt ] && [ ! ${ES_BEATS}/libbeat/tests/system/requirements.txt -ef ./tests/system/requirements.txt ] ; then \
		. ${PYTHON_ENV}/bin/activate && pip install -qUr ${ES_BEATS}/libbeat/tests/system/requirements.txt -Ur ./tests/system/requirements.txt ; \
	else \
		. ${PYTHON_ENV}/bin/activate && pip install -qUr ${ES_BEATS}/libbeat/tests/system/requirements.txt ; \
	fi


.PHONY: test
test: ## @testing Runs unit and system tests without coverage reports
test: unit
	if [ $(SYSTEM_TESTS) = true ]; then \
		 $(MAKE) fast-system-tests; \
	fi

.PHONY: testsuite
testsuite: ## @testing Runs all tests and generates the coverage reports
testsuite: clean collect
	$(MAKE) unit-tests

	# Setups environment if TEST_ENVIRONMENT is set to true
	# Only runs integration tests with test environemtn
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

	$(MAKE) benchmark-tests
	$(MAKE) coverage-report

	if [ $(TEST_ENVIRONMENT) = true ]; then \
		$(MAKE) fix-permissions; \
	fi

# Generates a coverage report from the existing coverage files
.PHONY: coverage-report
coverage-report:
	python ${ES_BEATS}/dev-tools/aggregate_coverage.py -o ${COVERAGE_DIR}/full.cov ${COVERAGE_DIR}
	go tool cover -html=${COVERAGE_DIR}/full.cov -o ${COVERAGE_DIR}/full.html
	test ! -s ${COVERAGE_DIR}/integration.cov   || go tool cover -html=${COVERAGE_DIR}/integration.cov   -o ${COVERAGE_DIR}/integration.html
	test ! -s ${COVERAGE_DIR}/system.cov || go tool cover -html=${COVERAGE_DIR}/system.cov -o ${COVERAGE_DIR}/system.html
	test ! -s ${COVERAGE_DIR}/unit.cov   || go tool cover -html=${COVERAGE_DIR}/unit.cov   -o ${COVERAGE_DIR}/unit.html


.PHONY: update
update: ## @build Update expects the most recent version of libbeat in the GOPATH
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

	# Generate index-pattern
	echo "Generate index pattern"
	-rm -f $(PWD)/_meta/kibana/index-pattern/${BEATNAME}.json
	mkdir -p $(PWD)/_meta/kibana/index-pattern
	. ${PYTHON_ENV}/bin/activate && python ${ES_BEATS}/libbeat/scripts/generate_index_pattern.py --index '${BEATNAME}-*' --libbeat ${ES_BEATS}/libbeat --beat $(PWD)

.PHONY: docs
docs:  ## @build Builds the documents for the beat
	sh ${ES_BEATS}/libbeat/scripts/build_docs.sh ${BEATNAME}


.PHONY: docs-preview
docs-preview:  ## @build Preview the documents for the beat in the browser
	if [ ! -d "build/docs" ]; then $(MAKE) docs; fi;
	${BUILD_DIR}/docs/build_docs.pl --chunk=1 -open chunk=1 -open --doc ${GOPATH}/src/github.com/elastic/beats/${BEATNAME}/docs/index.asciidoc -out ${BUILD_DIR}/html_docs


### KIBANA FILES HANDLING ###
ES_URL?=http://localhost:9200

.PHONY: export-dashboards
export-dashboards: python-env update
	. ${PYTHON_ENV}/bin/activate && python ${ES_BEATS}/dev-tools/export_dashboards.py --url ${ES_URL} --dir $(shell pwd)/_meta/kibana --regex ${BEATNAME}-*

${ES_BEATS}/libbeat/dashboards/import_dashboards:
	$(MAKE) -C ${ES_BEATS}/libbeat/dashboards import_dashboards

.PHONY: import-dashboards
import-dashboards: update ${ES_BEATS}/libbeat/dashboards/import_dashboards
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
	-${DOCKER_COMPOSE} rm -f -v

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
	if [ -a ${NOTICE_FILE} ]; then \
		install -m 644 ${NOTICE_FILE} ${HOME_PREFIX}/; \
	fi

# Prepares for packaging. Builds binaries and creates homedir data
.PHONY: prepare-package
prepare-package:
	# cross compile on ubuntu
	docker run --rm \
		-v $(abspath ${ES_BEATS}/dev-tools/packer/xgo-scripts):/scripts \
		-v $(abspath ${PACKER_TEMPLATES_DIR}):/templates \
		-v $(abspath ../):/source \
		-v $(BUILD_DIR):/build \
		-e PUREGO="yes" \
		-e PACK=${BEATNAME} \
		-e BEFORE_BUILD=before_build.sh \
		-e SOURCE=/source \
		-e TARGETS=${TARGETS} \
		-e BUILDID=${BUILDID} \
		${BEATS_BUILDER_IMAGE} \
		${BEAT_DIR}

# Prepares for packaging. Builds binaries with cgo
.PHONY: prepare-package-cgo
prepare-package-cgo:

	# cross compile on ubuntu
	docker run --rm \
		-v $(abspath ${ES_BEATS}/dev-tools/packer/xgo-scripts):/scripts \
		-v $(abspath ${PACKER_TEMPLATES_DIR}):/templates \
		-v $(abspath ../):/source \
		-v $(BUILD_DIR):/build \
		-e PACK=${BEATNAME} \
		-e BEFORE_BUILD=before_build.sh \
		-e SOURCE=/source \
		-e TARGETS=${TARGETS} \
		-e BUILDID=${BUILDID} \
		${BEATS_BUILDER_IMAGE} \
		${BEAT_DIR}

	# linux builds on debian 6 for compatibility
	docker run --rm \
		-v ${BUILD_DIR}:/build \
		-v $(abspath ${ES_BEATS}/dev-tools/packer/xgo-scripts):/scripts \
		-v $(abspath ${PACKER_TEMPLATES_DIR}):/templates \
		-v $(abspath ..):/source \
		-e PACK=${BEATNAME} \
		-e BEFORE_BUILD=before_build.sh \
		-e SOURCE=/source \
		-e TARGETS=${TARGETS_OLD} \
		-e BUILDID=${BUILDID} \
		${BEATS_BUILDER_DEB6_IMAGE} \
		${BEAT_DIR}

# Prepares images for packaging
.PHONY: package-setup
package-setup:
	$(MAKE) -C ${ES_BEATS}/dev-tools/packer deps images

.PHONY: package
package: ## @packaging Create binary packages for the beat.
package: package-setup

	echo "Start building packages for ${BEATNAME}"

	mkdir -p ${BUILD_DIR}/upload

	# Generates the package.yml file with all information needed to create packages
	echo "beat_name: ${BEATNAME}" > ${BUILD_DIR}/package.yml
	echo "beat_url: https://${BEAT_DIR}" >> ${BUILD_DIR}/package.yml
	echo "beat_repo: ${BEAT_DIR}" >> ${BUILD_DIR}/package.yml
	echo "beat_description: ${BEAT_DESCRIPTION}" >> ${BUILD_DIR}/package.yml
	echo "beat_vendor: ${BEAT_VENDOR}" >> ${BUILD_DIR}/package.yml
	echo "beat_license: ${BEAT_LICENSE}" >> ${BUILD_DIR}/package.yml
	echo "beat_doc_url: ${BEAT_DOC_URL}" >> ${BUILD_DIR}/package.yml

	if [ -a version.yml ]; then \
		cat version.yml >> ${BUILD_DIR}/package.yml; \
	else \
		cat ${ES_BEATS}/dev-tools/packer/version.yml >> ${BUILD_DIR}/package.yml; \
	fi

	if [ $(CGO) = true ]; then \
		 $(MAKE) prepare-package-cgo; \
	else \
		$(MAKE) prepare-package; \
	fi

	SNAPSHOT=${SNAPSHOT} BUILDID=${BUILDID} BEAT_DIR=${BEAT_DIR} BUILD_DIR=${BUILD_DIR} $(MAKE) -C ${ES_BEATS}/dev-tools/packer ${PACKAGES} ${BUILD_DIR}/upload/build_id.txt
	$(MAKE) fix-permissions
	echo "Finished packages for ${BEATNAME}"

package-dashboards: package-setup
	mkdir -p ${BUILD_DIR}
	cp -r _meta/kibana ${BUILD_DIR}/dashboards
	# build the dashboards package
	BEATNAME=${BEATNAME} BUILD_DIR=${BUILD_DIR} SNAPSHOT=$(SNAPSHOT) $(MAKE) -C ${ES_BEATS}/dev-tools/packer package-dashboards ${shell pwd}/build/upload/build_id.txt

fix-permissions:
	# Change ownership of all files inside /build folder from root/root to current user/group
	docker run -v ${BUILD_DIR}:/build alpine:3.4 sh -c "chown -R $(shell id -u):$(shell id -g) /build"

set_version: ## @packaging VERSION=x.y.z set the version of the beat to x.y.z
	${ES_BEATS}/dev-tools/set_version ${VERSION}

get_version: ## @packaging get the version of the beat.
	@${ES_BEATS}/dev-tools/get_version

help: ## @help Show this help.
	@python ${ES_BEATS}/libbeat/scripts/generate_makefile_doc.py $(MAKEFILE_LIST)

help_variables: ## @help Show Makefile customizable variables.
	@python ${ES_BEATS}/libbeat/scripts/generate_makefile_doc.py --variables $(MAKEFILE_LIST)
