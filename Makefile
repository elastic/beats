#/bin/bash

### VARIABLE SETUP ###

GODEP=$(GOPATH)/bin/godep
# Hidden directory to install dependencies for jenkins
export PATH := ./bin:$(PATH)
GOFILES = $(shell find . -type f -name '*.go')
SHELL=/bin/bash
ES_HOST?="elasticsearch-200"
BUILD_DIR=build
COVERAGE_DIR=${BUILD_DIR}/coverage
PROCESSES?= 4
TIMEOUT?= 90


### BUILDING ###

# Builds libbeat. No binary created as it is a library
.PHONY: build
build: deps
	$(GODEP) go build ./...

# Create test coverage binary
.PHONY: libbeat.test
libbeat.test: $(GOFILES)
	$(GODEP) go test -c -covermode=count -coverpkg ./...

# Cross-compile libbeat for the OS and architectures listed in
# crosscompile.bash. The binaries are placed in the ./bin dir.
.PHONY: crosscompile
crosscompile: $(GOFILES)
	go get github.com/tools/godep
	mkdir -p ${BUILD_DIR}/bin
	source scripts/crosscompile.bash; OUT='${BUILD_DIR}/bin' go-build-all

# Fetch dependencies
.PHONY: deps
deps:
	go get github.com/tools/godep
	# TODO: Is this still needed?
	go get -t ./...

# Checks project and source code if everything is according to standard
.PHONY: check
check:
	# This should be modified so it throws an error on the build system in case the output is not empty
	gofmt -d .
	godep go vet ./...

# Cleans up directory and source code with gofmt
.PHONY: clean
clean:
	go fmt ./...
	-rm -r build
	-rm libbeat.test

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
.PHONY: unit-tests
unit-tests: prepare-tests
	#go test -short ./...
	GOPATH=$(shell $(GODEP) path):$(GOPATH) $(GOPATH)/bin/gotestcover -coverprofile=${COVERAGE_DIR}/unit.cov -short -covermode=count github.com/elastic/libbeat/...

# Run integration tests. Unit tests are run as part of the integration tests
.PHONY: integration-tests
integration-tests: prepare-tests
	GOPATH=$(shell $(GODEP) path):$(GOPATH) $(GOPATH)/bin/gotestcover -coverprofile=${COVERAGE_DIR}/integration.cov -covermode=count github.com/elastic/libbeat/...

# Runs the integration inside a virtual environment. This can be run on any docker-machine (local, remote)
.PHONY: integration-tests-environment
integration-tests-environment:
	make prepare-tests
	make build-image
	NAME=$$(docker-compose run -d libbeat make integration-tests | awk 'END{print}') || exit 1; \
	echo "docker libbeat test container: '$$NAME'"; \
	docker attach $$NAME; CODE=$$?;\
	mkdir -p ${COVERAGE_DIR}; \
	docker cp $$NAME:/go/src/github.com/elastic/libbeat/${COVERAGE_DIR}/integration.cov $(shell pwd)/${COVERAGE_DIR}/; \
	docker rm $$NAME > /dev/null; \
	exit $$CODE

# Runs the system tests
.PHONY: system-tests
system-tests: libbeat.test prepare-tests system-tests-setup
	. build/system-tests/env/bin/activate; nosetests -w tests/system --processes=${PROCESSES} --process-timeout=$(TIMEOUT)
	# Writes count mode on top of file
	echo 'mode: count' > ${COVERAGE_DIR}/system.cov
	# Collects all system coverage files and skips top line with mode
	tail -q -n +2 ./build/system-tests/run/**/*.cov >> ${COVERAGE_DIR}/system.cov

# Runs the system tests
.PHONY: system-tests
system-tests-setup: tests/system/requirements.txt
	test -d env || virtualenv build/system-tests/env > /dev/null
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
	make integration-tests-environment
	make system-tests
	make benchmark-tests
	make coverage-report


# Generates a coverage report from the existing coverage files
# It assumes that some covrage reports already exists, otherwise it will fail
.PHONY: coverage-report
coverage-report:
	# Writes count mode on top of file
	echo 'mode: count' > ./${COVERAGE_DIR}/full.cov
	# Collects all coverage files and skips top line with mode
	tail -q -n +2 ./${COVERAGE_DIR}/*.cov >> ./${COVERAGE_DIR}/full.cov
	$(GODEP) go tool cover -html=./${COVERAGE_DIR}/full.cov -o ${COVERAGE_DIR}/full.html



### CONTAINER ENVIRONMENT ####

# Builds the environment to test libbeat
.PHONY: build-image
build-image: write-environment
	docker-compose build

# Runs the environment so the redis and elasticsearch can also be used for local development
# To use it for running the test, set ES_HOST and REDIS_HOST environment variable to the ip of your docker-machine.
.PHONY: start-environment
start-environment: stop-environment
	docker-compose up -d redis elasticsearch-173 elasticsearch-200 logstash

.PHONY: stop-environment
stop-environment:
	-docker-compose stop
	-docker-compose rm -f
	-docker ps -a  | grep libbeat | grep Exited | awk '{print $$1}' | xargs docker rm

.PHONY: write-environment
write-environment:
	mkdir -p build
	echo "ES_HOST=${ES_HOST}" > build/test.env
	echo "ES_PORT=9200" >> build/test.env
