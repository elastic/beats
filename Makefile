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

.PHONY: test
test:
	$(GODEP) go test ./...


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
