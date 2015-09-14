ARCH?=$(shell uname -m)
GODEP=$(GOPATH)/bin/godep
GOFILES = $(shell find . -type f -name '*.go')

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
	-rm filebeat
	-rm .filebeat
	-rm profile.cov
	-rm -r cover

.PHONY: run
run: filebeat
	./filebeat -c etc/filebeat.dev.yml -e -v -d "*"

.PHONY: test
test:
	$(GODEP) go test -short ./...

.PHONY: cover
cover:
	# gotestcover is needed to fetch coverage for multiple packages
	go get github.com/pierrre/gotestcover
	GOPATH=$(shell $(GODEP) path):$(GOPATH) $(GOPATH)/bin/gotestcover -coverprofile=profile.cov -covermode=count github.com/elastic/filebeat/...
	mkdir -p cover
	$(GODEP) go tool cover -html=profile.cov -o cover/coverage.html

# Command used by CI Systems
.PHONE: testsuite
testsuite: filebeat
	make cover

filebeat.test: $(GOFILES)
	$(GODEP) go test -c -cover -covermode=count -coverpkg ./...

full-coverage:
	make coverage
	make -C ./tests/integration coverage
	# Writes count mode on top of file
	echo 'mode: count' > ./coverage/full.cov
	# Collects all integration coverage files and skips top line with mode
	tail -q -n +2 ./coverage/*.cov >> ./coverage/full.cov
	$(GODEP) go tool cover -html=./coverage/full.cov -o coverage/full.html
