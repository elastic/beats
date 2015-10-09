GODEP=$(GOPATH)/bin/godep
# Hidden directory to install dependencies for jenkins
export PATH := ./bin:$(PATH)
GOFILES = $(shell find . -type f -name '*.go')
SHELL=/bin/bash
ES_HOST?="elasticsearch-200"

.PHONY: build
build:
	go get github.com/tools/godep
	$(GODEP) go build ./...

.PHONY: deps
deps:
	go get -t ./...
	# goautotest is used from the Makefile to run tests in a loop
	go get github.com/tsg/goautotest
	# cover
	go get golang.org/x/tools/cmd/cover

.PHONY: gofmt
gofmt:
	go fmt ./...

.PHONY: test
test:
	$(GODEP) go test -short ./...

.PHONY: check
check:
	# This should be modified so it throws an error on the build system in case the output is not empty
	gofmt -d .
	godep go vet ./...

.PHONY: autotest
autotest:
	goautotest -short ./...

.PHONY: testlong
testlong:
	go vet ./...
	make coverage

.PHONY: benchmark
benchmark:
	go test -short -bench=. ./...

.PHONY: coverage
coverage:
	# gotestcover is needed to fetch coverage for multiple packages
	go get github.com/pierrre/gotestcover
	mkdir -p coverage
	GOPATH=$(shell $(GODEP) path):$(GOPATH) $(GOPATH)/bin/gotestcover -coverprofile=coverage/unit.cov -covermode=count github.com/elastic/libbeat/...
	$(GODEP) go tool cover -html=coverage/unit.cov -o coverage/unit.html

.PHONY: clean
clean:
	make gofmt
	-rm -r coverage


# Builds the environment to test libbeat
.PHONY: build-image
build-image: write-environment
	make clean
	docker-compose build

# Runs the environment so the redis and elasticsearch can also be used for local development
# To use it for running the test, set ES_HOST and REDIS_HOST environment variable to the ip of your docker-machine.
.PHONY: start-environment
start-environment: stop-environment
	docker-compose up -d redis elasticsearch-172 elasticsearch-200 logstash

.PHONY: stop-environment
stop-environment:
	-docker-compose stop
	-docker-compose rm -f
	-docker ps -a  | grep libbeat | grep Exited | awk '{print $$1}' | xargs docker rm

.PHONY: write-environment
write-environment:
	echo "ES_HOST=${ES_HOST}" > docker/test.env
	echo "ES_PORT=9200" >> docker/test.env

# Runs the full test suite and puts out the result. This can be run on any docker-machine (local, remote)
.PHONY: testsuite
testsuite: build-image write-environment
	NAME=$$(docker-compose run -d libbeat make testlong) || exit 1; \
	docker attach $$NAME; CODE=$$?;\
	mkdir -p coverage; \
	docker cp $$NAME:/go/src/github.com/elastic/libbeat/coverage/unit.cov $(shell pwd)/coverage/; \
	docker cp $$NAME:/go/src/github.com/elastic/libbeat/coverage/unit.html $(shell pwd)/coverage/; \
	docker rm $$NAME > /dev/null; \
	exit $$CODE

# Sets up docker-compose locally for jenkins so no global installation is needed
.PHONY: docker-compose-setup
docker-compose-setup:
	mkdir -p bin
	curl -L https://github.com/docker/compose/releases/download/1.4.0/docker-compose-`uname -s`-`uname -m` > bin/docker-compose
	chmod +x bin/docker-compose

.PHONY: libbeat.test
libbeat.test: $(GOFILES)
	$(GODEP) go test -c -covermode=count -coverpkg ./...


.PHONY: system-tests
system-tests: libbeat.test
	mkdir -p coverage
	./libbeat.test -c tests/files/config.yml -d "*" -test.coverprofile coverage/system.cov

# Cross-compile libbeat for the OS and architectures listed in
# crosscompile.bash. The binaries are placed in the ./bin dir.
.PHONY: crosscompile
crosscompile: $(GOFILES)
	go get github.com/tools/godep
	curl https://raw.githubusercontent.com/elastic/filebeat/fe0f6a82d46b56d852f3f9ef81196aef4624d1a7/crosscompile.bash > crosscompile.bash
	mkdir -p bin
	source crosscompile.bash; OUT='bin' go-build-all

.PHONY: full-coverage
full-coverage:
	make testlong
	make -C ./tests/system coverage
	# Writes count mode on top of file
	echo 'mode: count' > ./coverage/full.cov
	# Collects all coverage files and skips top line with mode
	tail -q -n +2 ./coverage/*.cov >> ./coverage/full.cov
	$(GODEP) go tool cover -html=./coverage/full.cov -o coverage/full.html
