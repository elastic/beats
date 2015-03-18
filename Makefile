BIN_PATH?=/usr/bin
CONF_PATH?=/etc/packetbeat
VERSION?=0.4.3
ARCH?=$(shell uname -m)

GOFILES = $(shell find . -type f -name '*.go')
packetbeat: $(GOFILES)
	go build

go-daemon/god: go-daemon/god.c
	make -C go-daemon

.PHONY: with_pfring
with_pfring:
	go build --tags havepfring

.PHONY: deps
deps:
	go get
	# the testify one we need to spell out, because it's only used in tests
	go get github.com/stretchr/testify
	# make sure vet is installed
	go get golang.org/x/tools/cmd/vet
	# yamlcheck is used to verify the spec
	go get github.com/tsg/yamlcheck
	# goautotest is used from the Makefile to run tests in a loop
	go get github.com/tsg/goautotest
	# websocket is needed by the gobeacon tests
	go get golang.org/x/net/websocket

.PHONY: install
install: packetbeat go-daemon/god
	install -D packetbeat $(DESTDIR)/$(BIN_PATH)/packetbeat
	install -D go-daemon/god $(DESTDIR)/$(BIN_PATH)/packetbeat-god
	install -m 644 -D packetbeat.conf $(DESTDIR)/$(CONF_PATH)/packetbeat.conf

.PHONY: dist
dist: packetbeat go-daemon/god
	mkdir packetbeat-$(VERSION)
	cp packetbeat packetbeat-$(VERSION)/
	cp go-daemon/god packetbeat-$(VERSION)/packetbeat-god
	cp packetbeat.conf packetbeat-$(VERSION)/
	tar czvf packetbeat-$(VERSION)-$(ARCH).tar.gz packetbeat-$(VERSION)

.PHONY: gofmt
gofmt:
	go fmt ./...

.PHONY: test
test:
	go test -short ./...
	make -C tests test

.PHONY: autotest
autotest:
	goautotest -short ./...

.PHONY: testlong
testlong:
	go vet ./...
	go test ./...
	make -C tests test

.PHONY: cover
cover:
	go test -short -coverprofile=coverage.out
	go tool cover -html=coverage.out

.PHONY: benchmark
benchmark:
	go test -short -bench=. ./...

.PHONY: clean
clean:
	rm packetbeat || true
	rm -r packetbeat-$(VERSION) || true
