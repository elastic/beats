GODEP=$(GOPATH)/bin/godep

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

.PHONY: autotest
autotest:
	goautotest -short ./...

.PHONY: testlong
testlong:
	go vet ./...
	make cover

.PHONY: benchmark
benchmark:
	go test -short -bench=. ./...

.PHONY: cover
cover:
	# gotestcover is needed to fetch coverage for multiple packages
	go get github.com/pierrre/gotestcover
	GOPATH=$(shell $(GODEP) path):$(GOPATH) $(GOPATH)/bin/gotestcover -coverprofile=profile.cov -covermode=count github.com/elastic/libbeat/...
	mkdir -p cover
	$(GODEP) go tool cover -html=profile.cov -o cover/coverage.html

.PHONY: clean
clean:
	make gofmt
	-rm profile.cov
	-rm -r cover


# Builds the environment to test libbeat
.PHONY: build-image
build-image:
	make clean
	docker-compose build

# Runs the environment so the redis and elasticsearch can also be used for local development
# To use it for running the test, set ES_HOST and REDIS_HOST environment variable to the ip of your docker-machine.
.PHONY: start-environment
start-environment: build-image
	docker-compose up -d
	
.PHONY: stop-environment
stop-environment:
	docker-compose stop
	docker-compose rm -f

# Runs the full test suite and puts out the result. This can be run on any docker-machine (local, remote)
.PHONY: testsuite
testsuite: build-image
	docker-compose run libbeat make testlong
	# Copy coverage file back to host
	mkdir -p cover
	docker cp libbeat_libbeat_run_1:/go/src/github.com/elastic/libbeat/profile.cov $(shell pwd)/profile.cov
	docker cp libbeat_libbeat_run_1:/go/src/github.com/elastic/libbeat/cover/coverage.html $(shell pwd)/cover/coverage.html
	make stop-environment
