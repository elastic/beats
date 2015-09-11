ARCH?=$(shell uname -m)
GODEP=$(GOPATH)/bin/godep

.PHONY: build
build:
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
	-rm profile.cov
	-rm -r cover

.PHONY: run
run: build
	./filebeat -c etc/filebeat.yml -config etc/filebeat.yml -e -v

.PHONY: test
test:
	$(GODEP) go test -short ./...
	make -C tests test

.PHONY: cover
cover:
	# gotestcover is needed to fetch coverage for multiple packages
	go get github.com/pierrre/gotestcover
	GOPATH=$(shell $(GODEP) path):$(GOPATH) $(GOPATH)/bin/gotestcover -coverprofile=profile.cov -covermode=count github.com/elastic/filebeat/...
	mkdir -p cover
	$(GODEP) go tool cover -html=profile.cov -o cover/coverage.html
