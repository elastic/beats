### VARIABLE SETUP ###
### Application using libbeat may override the following variables in their Makefile
BEAT_NAME?=libbeat## @packaging Name of the binary
BEAT_TITLE?=${BEAT_NAME}## @packaging Title of the application
BEAT_DESCRIPTION?=Sends events to Elasticsearch or Logstash ## @packaging Description of the application
BEAT_PATH?=github.com/elastic/beats/${BEAT_NAME}
BEAT_PACKAGE_NAME?=${BEAT_NAME}
BEAT_INDEX_PREFIX?=${BEAT_NAME}
BEAT_URL?=https://www.elastic.co/products/beats/${BEAT_NAME} ## @packaging Link to the homepage of the application
BEAT_DOC_URL?=https://www.elastic.co/guide/en/beats/${BEAT_NAME}/current/index.html ## @packaging Link to the user documentation of the application
BEAT_LICENSE?=ASL 2.0 ## @packaging Software license of the application
BEAT_VENDOR?=Elastic ## @packaging Name of the vendor of the application
BEAT_GOPATH=$(firstword $(subst :, ,${GOPATH}))
ES_BEATS?=..## @community_beat Must be set to ./vendor/github.com/elastic/beats. It must always be a relative path.
GOPACKAGES?=$(shell go list ${BEAT_PATH}/... | grep -v /vendor/ | grep -v /scripts/cmd/ )
PACKER_TEMPLATES_DIR?=${ES_BEATS}/dev-tools/packer ## @Building Directory of templates that are used by "make package"
NOTICE_FILE?=../NOTICE.txt
LICENSE_FILE?=../licenses/APACHE-LICENSE-2.0.txt
ELASTIC_LICENSE_FILE?=../licenses/ELASTIC-LICENSE.txt

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
LIBCOMPOSE_TOOL_REPO?=github.com/docker/libcompose
NOW=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GOBUILD_FLAGS?=-i -ldflags "-X github.com/elastic/beats/libbeat/version.buildTime=$(NOW) -X github.com/elastic/beats/libbeat/version.commit=$(COMMIT_ID)"
GOIMPORTS=goimports
GOIMPORTS_REPO?=golang.org/x/tools/cmd/goimports
GOIMPORTS_LOCAL_PREFIX?=github.com/elastic
GOLINT=golint
GOLINT_REPO?=github.com/golang/lint/golint
REVIEWDOG?=reviewdog -conf ${ES_BEATS}/reviewdog.yml
REVIEWDOG_OPTIONS?=-diff "git diff master"
REVIEWDOG_REPO?=github.com/haya14busa/reviewdog/cmd/reviewdog
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
TESTING_ENVIRONMENT?=latest## @testing The name of the environment under test
BEAT_VERSION=$(shell head -n 1 ${ES_BEATS}/libbeat/docs/version.asciidoc | cut -c 17- )
COMMIT_ID=$(shell git rev-parse HEAD)
DOCKER_COMPOSE_PROJECT_NAME?=${BEAT_NAME}${TESTING_ENVIRONMENT}${COMMIT_ID} ## @testing The name of the docker-compose project used by the integration and system tests
DOCKER_COMPOSE?=TESTING_ENVIRONMENT=${TESTING_ENVIRONMENT} ES_BEATS=${ES_BEATS} docker-compose -p ${DOCKER_COMPOSE_PROJECT_NAME} -f docker-compose.yml
DOCKER_CACHE?=1 ## @miscellaneous If set to 0, all docker images are created without cache
GOPACKAGES_COMMA_SEP=$(subst $(space),$(comma),$(strip ${GOPACKAGES}))
PYTHON_ENV?=${BUILD_DIR}/python-env
PIP_INSTALL_PARAMS?=
BUILDID?=$(shell git rev-parse HEAD) ## @Building The build ID
VIRTUALENV_PARAMS?=
INTEGRATION_TESTS?=
FIND=. ${PYTHON_ENV}/bin/activate; find . -type f -not -path "*/vendor/*" -not -path "*/build/*" -not -path "*/.git/*"
PERM_EXEC?=$(shell [ `uname -s` = "Darwin" ] && echo "+111" || echo "/a+x")

# Cross compiling targets
CGO?=true ## @building if true, Build with C Go support
TARGETS?="windows/amd64 windows/386 darwin/amd64" ## @building list of platforms/architecture to be built by "make package"
TARGETS_OLD?="linux/amd64 linux/386" ## @building list of Debian6 architecture to be built by "make package" when CGO is true
PACKAGES?=${BEAT_NAME}/deb ${BEAT_NAME}/rpm ${BEAT_NAME}/darwin ${BEAT_NAME}/win ${BEAT_NAME}/bin ## @Building List of OS to be supported by "make package"
SNAPSHOT?=yes ## @Building If yes, builds a snapshot version
BEATS_BUILDER_IMAGE?=tudorg/beats-builder ## @Building Name of the docker image to use when packaging the application
BEATS_BUILDER_DEB_IMAGE?=tudorg/beats-builder-deb7 ## @Building Name of the docker image to use when packaging the application for Debian 7

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
	go get github.com/mitchellh/gox
	mkdir -p ${BUILD_DIR}/bin
	gox -output="${BUILD_DIR}/bin/{{.Dir}}-{{.OS}}-{{.Arch}}" -os="$(strip $(GOX_OS))" -osarch="$(strip $(GOX_OSARCH))" ${GOX_FLAGS}

.PHONY: check
check: python-env prepare-tests ## @build Checks project and source code if everything is according to standard
	@go vet ${GOPACKAGES}
	@go get $(GOIMPORTS_REPO)
	@goimports -local ${GOIMPORTS_LOCAL_PREFIX} -l ${GOFILES_NOVENDOR} | (! grep .) || (echo "Code differs from goimports' style ^" && false)
	@${FIND} -name *.py -exec autopep8 -d --max-line-length 120  {} \; | (! grep . -q) || (echo "Code differs from autopep8's style" && false)
	@${FIND} -wholename "*tests/system/test_*.py" -perm ${PERM_EXEC} -exec false {} + || (echo "Python test files shouldn't be executable, otherwise nose doesn't find them" && false)

.PHONY: fmt
fmt: python-env ## @build Runs `goimports -l -w` and `autopep8`on the project's source code, modifying any files that do not match its style.
	@go get $(GOIMPORTS_REPO)
	@goimports -local ${GOIMPORTS_LOCAL_PREFIX} -l -w ${GOFILES_NOVENDOR}
	@${FIND} -name *.py -exec ${PYTHON_ENV}/bin/autopep8 --in-place --max-line-length 120  {} \;

.PHONY: lint
lint:
	@go get $(GOLINT_REPO) $(REVIEWDOG_REPO)
	$(REVIEWDOG) $(REVIEWDOG_OPTIONS)

.PHONY: clean
clean:: ## @build  Cleans up all files generated by the build steps
	@rm -rf build
	@rm -f docker-compose.yml.lock
	@rm -f ${BEAT_NAME} ${BEAT_NAME}.test ${BEAT_NAME}.exe ${BEAT_NAME}.test.exe
	@rm -f _meta/fields.generated.yml fields.yml
	@rm -f $(PWD)/_meta/kibana/5/index-pattern/${BEAT_NAME}.json
	@rm -f $(PWD)/_meta/kibana/6/index-pattern/${BEAT_NAME}.json
	@rm -f ${BEAT_NAME}.template*.json

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
	# testify is needed for unit and integration tests
	go get ${TESTIFY_TOOL_REPO}
	# libcompose is needed for integration tests
	go get ${LIBCOMPOSE_TOOL_REPO}

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
	${DOCKER_COMPOSE} run beat make integration-tests RACE_DETECTOR=$(RACE_DETECTOR) DOCKER_COMPOSE_PROJECT_NAME=${DOCKER_COMPOSE_PROJECT_NAME}

# Runs the system tests
.PHONY: system-tests
system-tests: ## @testing Runs the system tests
system-tests: prepare-tests ${BEAT_NAME}.test python-env ${ES_BEATS}/dev-tools/cmd/dashboards/export_dashboards
	. ${PYTHON_ENV}/bin/activate; INTEGRATION_TESTS=${INTEGRATION_TESTS} TESTING_ENVIRONMENT=${TESTING_ENVIRONMENT} DOCKER_COMPOSE_PROJECT_NAME=${DOCKER_COMPOSE_PROJECT_NAME} nosetests ${PYTHON_TEST_FILES} ${NOSETESTS_OPTIONS}
	python ${ES_BEATS}/dev-tools/aggregate_coverage.py -o ${COVERAGE_DIR}/system.cov ./build/system-tests/run

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
	@test -d ${PYTHON_ENV} || virtualenv ${VIRTUALENV_PARAMS} ${PYTHON_ENV}
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


.PHONY: update
update: ## @build Update expects the most recent version of libbeat in the GOPATH
update: python-env collect
	@echo "Updating generated files for ${BEAT_NAME}"

	@# Update config files.
	@cat _meta/beat.yml ${ES_BEATS}/libbeat/_meta/config.yml | sed -e "s/beatname/${BEAT_NAME}/g;s/beat-index-prefix/${BEAT_INDEX_PREFIX}/g" > ${BEAT_NAME}.yml
	@if [ -e _meta/beat.reference.yml ]; then \
        cat _meta/beat.reference.yml ${ES_BEATS}/libbeat/_meta/config.reference.yml | sed -e "s/beatname/${BEAT_NAME}/g;s/beat-index-prefix/${BEAT_INDEX_PREFIX}/g" > ${BEAT_NAME}.reference.yml ; \
    else \
        cat _meta/beat.yml ${ES_BEATS}/libbeat/_meta/config.reference.yml | sed -e "s/beatname/${BEAT_NAME}/g;s/beat-index-prefix/${BEAT_INDEX_PREFIX}/g" > ${BEAT_NAME}.reference.yml ; \
    fi;
	@chmod 0640 ${BEAT_NAME}.yml ${BEAT_NAME}.reference.yml

	@# Update libbeat fields.generated.yml
	@$(MAKE) -C ${ES_BEATS}/libbeat fields

	@# Update fields.yml
	@test -s  _meta/fields.generated.yml || cp _meta/fields.yml _meta/fields.generated.yml
ifeq ($(BEAT_NAME), libbeat)
	@cp ${ES_BEATS}/libbeat/_meta/fields.generated.yml fields.yml
else
	@cat ${ES_BEATS}/libbeat/_meta/fields.generated.yml _meta/fields.generated.yml > fields.yml
endif

	@# Update docs
	@mkdir -p docs
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/libbeat/scripts/generate_fields_docs.py $(PWD) ${BEAT_TITLE} ${ES_BEATS}

	@# Generate Kibana index pattern
	@mkdir -p $(PWD)/_meta/kibana/5/index-pattern
	@mkdir -p $(PWD)/_meta/kibana/6/index-pattern
	@go run ${ES_BEATS}/dev-tools/cmd/kibana_index_pattern/kibana_index_pattern.go -index '${BEAT_INDEX_PREFIX}-*' -beat-name ${BEAT_NAME} -beat-dir $(PWD) -version ${BEAT_VERSION}

.PHONY: docs
docs:  ## @build Builds the documents for the beat
	sh ${ES_BEATS}/script/build_docs.sh ${BEAT_NAME} ${BEAT_PATH}/docs ${BUILD_DIR}


.PHONY: docs-preview
docs-preview:  ## @build Preview the documents for the beat in the browser
	PREVIEW=1 $(MAKE) docs

.PHONY: index-template
index-template: ## @build Generate index templates for the given $VERSION. This is for manual testing.
	go run ${ES_BEATS}/dev-tools/cmd/index_template/index_template.go -index ${BEAT_INDEX_PREFIX} -output ${BEAT_GOPATH}/src/${BEAT_PATH}/${BEAT_NAME}.template-es${VERSION}.json -file ${BEAT_GOPATH}/src/${BEAT_PATH}/fields.yml -es.version=${VERSION}

### KIBANA FILES HANDLING ###
ES_URL?=http://localhost:9200
KIBANA_URL?=http://localhost:5601

${ES_BEATS}/dev-tools/cmd/dashboards/export_dashboards:
	$(MAKE) -C ${ES_BEATS}/dev-tools/cmd/dashboards export_dashboards

.PHONY: import-dashboards
import-dashboards: update ${BEAT_NAME}
	${BEAT_GOPATH}/src/${BEAT_PATH}/${BEAT_NAME} setup -E setup.dashboards.directory=${PWD}/_meta/kibana -E setup.kibana.host=${KIBANA_URL} --dashboards

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


### Packaging targets ####

# Installs the files that need to get to the home path on installations
HOME_PREFIX?=/tmp/${BEAT_NAME}
.PHONY: install-home
install-home:
	if [ -a ${NOTICE_FILE} ]; then \
		install -m 644 ${NOTICE_FILE} ${HOME_PREFIX}/; \
	fi
	if [ -a ${LICENSE_FILE} ]; then \
		install -m 644 ${LICENSE_FILE} ${HOME_PREFIX}/LICENSE.txt; \
	fi
	if [ -d _meta/module.generated ]; then \
		install -d -m 755 ${HOME_PREFIX}/module; \
		rsync -av _meta/module.generated/ ${HOME_PREFIX}/module/; \
		chmod -R go-w ${HOME_PREFIX}/module/; \
	fi
	if [ -d _meta/kibana ]; then \
		install -d -m 755 ${HOME_PREFIX}/kibana; \
		rsync -av _meta/kibana/ ${HOME_PREFIX}/kibana/; \
	fi

# Prepares for packaging. Builds binaries and creates homedir data
.PHONY: prepare-package
prepare-package:
	# cross compile on ubuntu
	docker run --rm \
		-v $(abspath ${ES_BEATS}/dev-tools/packer/xgo-scripts):/scripts \
		-v $(abspath ${PACKER_TEMPLATES_DIR}):/templates \
		-v $(abspath ../):/source \
		-v $(PKG_BUILD_DIR):/build \
		-e PUREGO="yes" \
		-e PACK=${BEAT_NAME} \
		-e BEFORE_BUILD=before_build.sh \
		-e SOURCE=/source \
		-e TARGETS=${TARGETS} \
		-e BUILDID=${BUILDID} \
		-e ES_BEATS=${ES_BEATS} \
		-e BEAT_PATH=${BEAT_PATH} \
		-e BEAT_NAME=${BEAT_NAME} \
		-e LICENSE_FILE=${LICENSE_FILE} \
		${BEATS_BUILDER_IMAGE}

# Prepares for packaging. Builds binaries with cgo
.PHONY: prepare-package-cgo
prepare-package-cgo:

	# cross compile on ubuntu
	docker run --rm \
		-v $(abspath ${ES_BEATS}/dev-tools/packer/xgo-scripts):/scripts \
		-v $(abspath ${PACKER_TEMPLATES_DIR}):/templates \
		-v $(abspath ../):/source \
		-v $(PKG_BUILD_DIR):/build \
		-e PACK=${BEAT_NAME} \
		-e BEFORE_BUILD=before_build.sh \
		-e SOURCE=/source \
		-e TARGETS=${TARGETS} \
		-e BUILDID=${BUILDID} \
		-e ES_BEATS=${ES_BEATS} \
		-e BEAT_PATH=${BEAT_PATH} \
		-e BEAT_NAME=${BEAT_NAME} \
		-e LICENSE_FILE=${LICENSE_FILE} \
		${BEATS_BUILDER_IMAGE}

	# linux builds on older debian for compatibility
	docker run --rm \
		-v $(abspath ${ES_BEATS}/dev-tools/packer/xgo-scripts):/scripts \
		-v $(abspath ${PACKER_TEMPLATES_DIR}):/templates \
		-v $(abspath ..):/source \
		-v ${PKG_BUILD_DIR}:/build \
		-e PACK=${BEAT_NAME} \
		-e BEFORE_BUILD=before_build.sh \
		-e SOURCE=/source \
		-e TARGETS=${TARGETS_OLD} \
		-e BUILDID=${BUILDID} \
		-e ES_BEATS=${ES_BEATS} \
		-e BEAT_PATH=${BEAT_PATH} \
		-e BEAT_NAME=${BEAT_NAME} \
		-e LICENSE_FILE=${LICENSE_FILE} \
		${BEATS_BUILDER_DEB_IMAGE}

# Prepares images for packaging
.PHONY: package-setup
package-setup:
	$(MAKE) -C ${ES_BEATS}/dev-tools/packer deps images

.PHONY: package
package: ## @packaging Create binary packages for the beat.
package: update package-setup
	echo "Start building packages for ${BEAT_NAME}"

	rm -rf ${PKG_BUILD_DIR}
	mkdir -p ${PKG_BUILD_DIR}
	mkdir -p ${PKG_UPLOAD_DIR}

	# Generates the package.yml file with all information needed to create packages
	echo "beat_name: ${BEAT_NAME}"                          >  ${PKG_BUILD_DIR}/package.yml
	echo "beat_url: ${BEAT_URL}"                            >> ${PKG_BUILD_DIR}/package.yml
	echo "beat_repo: ${BEAT_PATH}"                          >> ${PKG_BUILD_DIR}/package.yml
	echo "beat_pkg_name: ${BEAT_PACKAGE_NAME}"              >> ${PKG_BUILD_DIR}/package.yml
	echo "beat_pkg_suffix: '${PKG_SUFFIX}'"                 >> ${PKG_BUILD_DIR}/package.yml
	echo "beat_description: ${BEAT_DESCRIPTION}"            >> ${PKG_BUILD_DIR}/package.yml
	echo "beat_vendor: ${BEAT_VENDOR}"                      >> ${PKG_BUILD_DIR}/package.yml
	echo "beat_license: ${BEAT_LICENSE}"                    >> ${PKG_BUILD_DIR}/package.yml
	echo "beat_doc_url: ${BEAT_DOC_URL}"                    >> ${PKG_BUILD_DIR}/package.yml

	if [ -a version.yml ]; then \
		cat version.yml >> ${PKG_BUILD_DIR}/package.yml; \
	else \
		cat ${ES_BEATS}/dev-tools/packer/version.yml >> ${PKG_BUILD_DIR}/package.yml; \
	fi

	if [ $(CGO) = true ]; then \
		 $(MAKE) prepare-package-cgo; \
	else \
		$(MAKE) prepare-package; \
	fi

	SNAPSHOT=${SNAPSHOT} BUILDID=${BUILDID} BEAT_PATH=${BEAT_PATH} BUILD_DIR=${PKG_BUILD_DIR} UPLOAD_DIR=${PKG_UPLOAD_DIR} $(MAKE) -C ${ES_BEATS}/dev-tools/packer ${PACKAGES} ${BUILD_DIR}/upload/build_id.txt
	$(MAKE) fix-permissions
	echo "Finished packages for ${BEAT_NAME}"

# Packages the Beat without Elastic X-Pack content (OSS only).
.PHONY: package-oss
package-oss:
	@$(MAKE) PKG_SUFFIX=-oss package

# Packages the Beat with Elastic X-Pack content.
.PHONY: package-elastic
package-elastic:
	@$(MAKE) BEAT_LICENSE="Elastic License" LICENSE_FILE=$(ELASTIC_LICENSE_FILE) package

.PHONY: package-all
package-all: package-elastic package-oss

package-dashboards: package-setup
	mkdir -p ${BUILD_DIR}
	cp -r _meta/kibana ${BUILD_DIR}/dashboards
	# build the dashboards package
	BEAT_NAME=${BEAT_NAME} BUILD_DIR=${BUILD_DIR} SNAPSHOT=$(SNAPSHOT) $(MAKE) -C ${ES_BEATS}/dev-tools/packer package-dashboards ${shell pwd}/build/upload/build_id.txt

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
