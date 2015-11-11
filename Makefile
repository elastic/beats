GODEP=$(GOPATH)/bin/godep
PREFIX?=/build

GOFILES = $(shell find . -type f -name '*.go')
topbeat: $(GOFILES)
	# first make sure we have godep
	go get github.com/tools/godep
	$(GODEP) go build

.PHONY: getdeps
getdeps:
	go get -t -u -f

.PHONY: unit
unit:
	$(GODEP) go test ./...

.PHONY: test
test: unit
	make -C ./tests/system test

topbeat.test: $(GOFILES)
	$(GODEP) go test -c -covermode=count -coverpkg ./...

.PHONY: coverage
coverage:
	# gotestcover is needed to fetch coverage for multiple packages
	go get github.com/pierrre/gotestcover
	mkdir -p coverage
	GOPATH=$(shell $(GODEP) path):$(GOPATH) $(GOPATH)/bin/gotestcover -race -coverprofile=coverage/unit.cov -covermode=atomic github.com/elastic/topbeat/...

.PHONY: full-coverage
full-coverage:
	make coverage
	make -C ./tests/system coverage
	# Writes count mode on top of file
	echo 'mode: count' > ./coverage/full.cov
	# Collects all coverage files and skips top line with mode
	tail -q -n +2 ./coverage/*.cov >> ./coverage/full.cov
	$(GODEP) go tool cover -html=./coverage/full.cov -o coverage/full.html

# Command used by CI Systems
.PHONY: testsuite
testsuite: topbeat
	make coverage

.PHONY: install-cfg
install-cfg:
	cp etc/topbeat.template.json $(PREFIX)/topbeat.template.json
	# linux
	cp etc/topbeat.yml $(PREFIX)/topbeat-linux.yml
	# binary
	cp etc/topbeat.yml $(PREFIX)/topbeat-binary.yml
	# darwin
	cp etc/topbeat.yml $(PREFIX)/topbeat-darwin.yml
	# win
	cp etc/topbeat.yml $(PREFIX)/topbeat-win.yml

.PHONY: cover
cover:
	# gotestcover is needed to fetch coverage for multiple packages
	go get github.com/pierrre/gotestcover
	GOPATH=$(shell $(GODEP) path):$(GOPATH) $(GOPATH)/bin/gotestcover -race -coverprofile=profile.cov -covermode=atomic github.com/elastic/topbeat/...
	mkdir -p cover
	$(GODEP) go tool cover -html=profile.cov -o cover/coverage.html

.PHONY: clean
clean:
	rm -rf topbeat cover
