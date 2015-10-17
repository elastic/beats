ARCH?=$(shell uname -m)
GODEP=$(GOPATH)/bin/godep
GOFILES = $(shell find . -type f -name '*.go')
SHELL=/bin/bash

# default install folder used by the beats-packer
PREFIX?=/build

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

# This is called by the beats-packer to obtain the configuration file and
# default template
.PHONY: install-cfg
install-cfg:
	cp etc/filebeat.template.json $(PREFIX)/filebeat.template.json
	# linux
	cp etc/filebeat.yml $(PREFIX)/filebeat-linux.yml
	sed -i 's@#registry_file: .filebeat@registry_file: /var/lib/filebeat/registry@' $(PREFIX)/filebeat-linux.yml
	# binary
	cp etc/filebeat.yml $(PREFIX)/filebeat-binary.yml
	# darwin
	cp etc/filebeat.yml $(PREFIX)/filebeat-darwin.yml
	# win
	cp etc/filebeat.yml $(PREFIX)/filebeat-win.yml
	sed -i 's@#registry_file: .filebeat@registry_file: "C:/ProgramData/\filebeat/registry"@' $(PREFIX)/filebeat-win.yml
