ARCH?=$(shell uname -m)
SHELL=/bin/bash

.PHONY: build
build:
	GOOS=windows GOARCH=386 godep go build

.PHONY: native
native:
	godep go build

.PHONY: gen
gen: 
	GOOS=windows GOARCH=386 godep go generate -v -x ./...

.PHONY: check
check:
	gofmt -l . | read && echo "Code differs from gofmt's style" && exit 1 || true
	godep go vet ./...

.PHONY: clean
clean:
	gofmt -w .
	-rm -rf winlogbeat winlogbeat.exe winlogbeat.test .winlogbeat profile.cov coverage bin

.PHONY: unit
unit:
	godep go test ./...

.PHONY: coverage
coverage:
	mkdir -p coverage
	GOPATH=$(shell godep path):$(GOPATH) gotestcover -coverprofile=coverage/unit.cov -covermode=count github.com/elastic/winlogbeat/...
	godep go tool cover -html=coverage/unit.cov -o coverage/unit.html

.PHONY: install-deps
install-deps:
	go get github.com/tools/godep
	# gotestcover is needed to fetch coverage for multiple packages
	go get github.com/pierrre/gotestcover

.PHONY: update-deps
update-deps:
	godep update github.com/elastic/libbeat/...
