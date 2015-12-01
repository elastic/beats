
### VARIABLE SETUP ###

ARCH?=$(shell uname -m)
GODEP=$(GOPATH)/bin/godep
# Hidden directory to install dependencies for jenkins
export PATH := ./bin:$(PATH)
GOFILES = $(shell find . -type f -name '*.go')
SHELL=/bin/bash
ES_HOST?="elasticsearch-210"
BUILD_DIR=build
COVERAGE_DIR=${BUILD_DIR}/coverage
TIMEOUT?= 90
BEATNAME?=libbeat
TEST_ENVIRONMENT?=false
SYSTEM_TESTS?=false


### BUILDING ###

# Builds beat
.PHONY: build
build: deps $(GOFILES)
	$(GODEP) go build

# Create test coverage binary
.PHONY: libbeat.test
libbeat.test: $(GOFILES)
	$(GODEP) go test -c -covermode=atomic -coverpkg ./...

# Cross-compile beat for the OS and architectures listed in
# crosscompile.bash. The binaries are placed in the ./bin dir.
.PHONY: crosscompile
crosscompile: deps $(GOFILES)
	mkdir -p ${BUILD_DIR}/bin
	source ./scripts/crosscompile.bash; OUT='${BUILD_DIR}/bin' go-build-all

# Fetch dependencies
.PHONY: deps
deps:
	go get github.com/tools/godep

# Checks project and source code if everything is according to standard
.PHONY: check
check:
	gofmt -l . | read && echo "Code differs from gofmt's style" && exit 1 || true
	godep go vet ./...

# Cleans up directory and source code with gofmt
.PHONY: clean
clean:
	go fmt ./...
	gofmt -w .
	-rm -r build
	-rm ${BEATNAME} ${BEATNAME}.test

# Shortcut for continuous integration
# This should always run before merging.
.PHONY: ci
ci:
	make
	make check
	make testsuite

### Testing ###
# All tests are always run with coverage reporting enabled


# Prepration for tests
.PHONY: prepare-tests
prepare-tests:
	mkdir -p ${COVERAGE_DIR}
	# coverage tools
	go get golang.org/x/tools/cmd/cover
	# gotestcover is needed to fetch coverage for multiple packages
	go get github.com/pierrre/gotestcover

# Runs the unit tests
# Race is not enabled for unit tests because tests run much slower.
.PHONY: unit-tests
unit-tests: prepare-tests
	#go test -short ./...
	GOPATH=$(shell $(GODEP) path):$(GOPATH) $(GOPATH)/bin/gotestcover -coverprofile=${COVERAGE_DIR}/unit.cov -short -covermode=atomic github.com/elastic/${BEATNAME}/...

# Run integration tests. Unit tests are run as part of the integration tests. It runs all tests with race detection enabled.
.PHONY: integration-tests
integration-tests: prepare-tests
	GOPATH=$(shell $(GODEP) path):$(GOPATH) $(GOPATH)/bin/gotestcover -race -coverprofile=${COVERAGE_DIR}/integration.cov -covermode=atomic github.com/elastic/${BEATNAME}/...

# Runs the integration inside a virtual environment. This can be run on any docker-machine (local, remote)
.PHONY: integration-tests-environment
integration-tests-environment:
	make prepare-tests
	make build-image
	NAME=$$(docker-compose run -d ${BEATNAME} make integration-tests | awk 'END{print}') || exit 1; \
	echo "docker ${BEATNAME} test container: '$$NAME'"; \
	docker attach $$NAME; CODE=$$?;\
	mkdir -p ${COVERAGE_DIR}; \
	docker cp $$NAME:/go/src/github.com/elastic/${BEATNAME}/${COVERAGE_DIR}/integration.cov $(shell pwd)/${COVERAGE_DIR}/; \
	docker rm $$NAME > /dev/null; \
	exit $$CODE

# Runs the system tests
.PHONY: system-tests
system-tests: libbeat.test prepare-tests system-tests-setup
	. build/system-tests/env/bin/activate; nosetests -w tests/system --process-timeout=$(TIMEOUT) --with-timer
	# Writes atomic mode on top of file
	echo 'mode: atomic' > ${COVERAGE_DIR}/system.cov
	# Collects all system coverage files and skips top line with mode
	tail -q -n +2 ./build/system-tests/run/**/*.cov >> ${COVERAGE_DIR}/system.cov

# Runs the system tests
.PHONY: system-tests-setup
system-tests-setup: tests/system/requirements.txt
	test -d build/system-tests/env || virtualenv build/system-tests/env > /dev/null
	. build/system-tests/env/bin/activate && pip install -Ur tests/system/requirements.txt > /dev/null
	touch build/system-tests/env/bin/activate


# Run benchmark tests
.PHONY: benchmark-tests
benchmark-tests:
	# No benchmark tests exist so far
	#go test -short -bench=. ./...

# Runs all tests and generates the coverage reports
.PHONY: testsuite
testsuite:
	# Setups environment if TEST_ENVIRONMENT is set to true
	if [ $(TEST_ENVIRONMENT) = true ]; then \
		 make integration-tests-environment; \
	else \
		make integration-tests; \
	fi

	# Runs system tests if SYSTEM_TESTS is set to true
	if [ $(SYSTEM_TESTS) = true ]; then \
		 make system-tests; \
	fi

	make benchmark-tests
	make coverage-report

# Generates a coverage report from the existing coverage files
# It assumes that some covrage reports already exists, otherwise it will fail
.PHONY: coverage-report
coverage-report:
	# Writes atomic mode on top of file
	echo 'mode: atomic' > ./${COVERAGE_DIR}/full.cov
	# Collects all coverage files and skips top line with mode
	tail -q -n +2 ./${COVERAGE_DIR}/*.cov >> ./${COVERAGE_DIR}/full.cov
	$(GODEP) go tool cover -html=./${COVERAGE_DIR}/full.cov -o ${COVERAGE_DIR}/full.html

# Update expects the most recent version of libbeat in the GOPATH
.PHONY: update
update:
	bash ${GOPATH}/src/github.com/elastic/libbeat/scripts/update.sh ${BEATNAME} github.com/elastic/${BEATNAME}



### CONTAINER ENVIRONMENT ####

# Builds the environment to test beat
.PHONY: build-image
build-image: write-environment
	docker-compose build

# Runs the environment so the redis and elasticsearch can also be used for local development
# To use it for running the test, set ES_HOST and REDIS_HOST environment variable to the ip of your docker-machine.
.PHONY: start-environment
start-environment: stop-environment
	docker-compose up -d redis elasticsearch-173 elasticsearch-210 logstash

.PHONY: stop-environment
stop-environment:
	-docker-compose stop
	-docker-compose rm -f
	-docker ps -a  | grep ${BEATNAME} | grep Exited | awk '{print $$1}' | xargs docker rm

.PHONY: write-environment
write-environment:
	mkdir -p build
	echo "ES_HOST=${ES_HOST}" > build/test.env
	echo "ES_PORT=9200" >> build/test.env
	echo "ES_USER=beats" >> build/test.env
	echo "ES_PASS=testing" >> build/test.env
