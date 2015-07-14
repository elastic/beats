BIN_PATH?=/usr/bin
CONF_PATH?=/etc/packetbeat
VERSION?=1.0.0-beta2
ARCH?=$(shell uname -m)

GOFILES = $(shell find . -type f -name '*.go')
packetbeat: $(GOFILES)
	# first make sure we have godep
	go get github.com/tools/godep
	$(GOPATH)/bin/godep go build

go-daemon/god: go-daemon/god.c
	make -C go-daemon

.PHONY: with_pfring
with_pfring:
	go build --tags havepfring

.PHONY: getdeps
getdeps:
	go get -t -u -f
	# goautotest is used from the Makefile to run tests in a loop
	go get -u github.com/tsg/goautotest
	# websocket is needed by the gobeacon tests
	go get -u golang.org/x/net/websocket
	# godep is needed in this makefile
	go get -u github.com/tools/godep

.PHONY: deps
deps:
	# no longer needed
	true


.PHONY: updatedeps
updatedeps:
	godep update ...

.PHONY: install
install: packetbeat go-daemon/god
	install -D packetbeat $(DESTDIR)/$(BIN_PATH)/packetbeat
	install -D go-daemon/god $(DESTDIR)/$(BIN_PATH)/packetbeat-god
	install -m 644 -D etc/packetbeat.yml $(DESTDIR)/$(CONF_PATH)/packetbeat.yml
	install -m 644 -D etc/packetbeat.template.json $(DESTDIR)/$(CONF_PATH)/packetbeat.template.json

.PHONY: dist
dist: packetbeat go-daemon/god
	mkdir packetbeat-$(VERSION)
	cp packetbeat packetbeat-$(VERSION)/
	cp go-daemon/god packetbeat-$(VERSION)/packetbeat-god
	cp etc/packetbeat.yml packetbeat-$(VERSION)/
	cp etc/packetbeat.template.json packetbeat-$(VERSION)/
	tar czvf packetbeat-$(VERSION)-$(ARCH).tar.gz packetbeat-$(VERSION)

.PHONY: darwin_dist
darwin_dist: packetbeat
	mkdir packetbeat-$(VERSION)-darwin
	cp packetbeat packetbeat-$(VERSION)-darwin
	cp etc/packetbeat.yml packetbeat-$(VERSION)-darwin/
	cp etc/packetbeat.template.json packetbeat-$(VERSION)-darwin/
	sed -i .bk 's/device: any/device: en0/' packetbeat-$(VERSION)-darwin/packetbeat.yml
	rm packetbeat-$(VERSION)-darwin/packetbeat.yml.bk
	tar czvf packetbeat-$(VERSION)-darwin.tgz packetbeat-$(VERSION)-darwin
	shasum packetbeat-$(VERSION)-darwin.tgz > packetbeat-$(VERSION)-darwin.tgz.sha1.txt

.PHONY: gofmt
gofmt:
	go fmt ./...

.PHONY: test
test:
	$(GOPATH)/bin/godep go test -short ./...
	make -C tests test

.PHONY: autotest
autotest:
	goautotest -short ./...

.PHONY: testlong
testlong:
	go vet ./...
	$(GOPATH)/bin/godep go test ./...
	make -C tests test

.PHONY: cover
cover:
	mkdir -p cover
	./scripts/coverage.sh
	$(GOPATH)/bin/godep go tool cover -html=profile.cov -o cover/coverage.html

.PHONY: benchmark
benchmark:
	$(GOPATH)/bin/godep go test -short -bench=. ./...

.PHONY: gen
gen:
	./scripts/generate_gettingstarted.sh docs/gettingstarted.in.asciidoc docs/gettingstarted.asciidoc
	python scripts/generate_template.py etc/fields.yml etc/packetbeat.template.json
	python scripts/generate_field_docs.py etc/fields.yml docs/fields.asciidoc

.PHONY: clean
clean:
	rm packetbeat || true
	rm -r packetbeat-$(VERSION) || true
