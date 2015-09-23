ARCH?=$(shell uname -m)
GODEP=$(GOPATH)/bin/godep
GOFILES = $(shell find . -type f -name '*.go')
SHELL=/bin/bash

filebeat: $(GOFILES)
	# first make sure we have godep
	go get github.com/tools/godep
	$(GODEP) go build

.PHONY: check
check:
	# This should be modified so it throws an error on the build system in case the output is not empty
	gofmt -d .
	godep go vet ./...

.PHONY: clean
clean:
	gofmt -w .
	-rm -rf filebeat filebeat.test .filebeat profile.cov coverage bin

.PHONY: run
run: filebeat
	./filebeat -c etc/filebeat.dev.yml -e -v -d "*"

.PHONY: unit
unit:
	$(GODEP) go test ./...

.PHONY: test
test: unit
	make -C ./tests/system test

.PHONY: coverage
coverage:
	# gotestcover is needed to fetch coverage for multiple packages
	go get github.com/pierrre/gotestcover
	mkdir -p coverage
	GOPATH=$(shell $(GODEP) path):$(GOPATH) $(GOPATH)/bin/gotestcover -coverprofile=coverage/unit.cov -covermode=count github.com/elastic/filebeat/...
	$(GODEP) go tool cover -html=coverage/unit.cov -o coverage/unit.html

# Command used by CI Systems
.PHONY: testsuite
testsuite: filebeat
	make coverage

filebeat.test: $(GOFILES)
	$(GODEP) go test -c -covermode=count -coverpkg ./...

.PHONY: full-coverage
full-coverage:
	make coverage
	make -C ./tests/system coverage
	# Writes count mode on top of file
	echo 'mode: count' > ./coverage/full.cov
	# Collects all coverage files and skips top line with mode
	tail -q -n +2 ./coverage/*.cov >> ./coverage/full.cov
	$(GODEP) go tool cover -html=./coverage/full.cov -o coverage/full.html

# Cross-compile filebeat for the OS and architectures listed in
# crosscompile.bash. The binaries are placed in the ./bin dir.
.PHONY: crosscompile
crosscompile: $(GOFILES)
	go get github.com/tools/godep
	mkdir -p bin
	source crosscompile.bash; OUT='bin' go-build-all
