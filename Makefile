BIN_PATH?=/usr/bin
CONF_PATH?=/etc/packetbeat
VERSION?=0.3.2
ARCH?=$(shell uname -m)

packetbeat: *.go
	go build

go-daemon/god: go-daemon/god.c
	make -C go-daemon

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
	gofmt -w -tabs=false -tabwidth=4 *.go

.PHONY: test
test:
	go test -short

.PHONY: cover
cover:
	go test -short -coverprofile=coverage.out
	go tool cover -html=coverage.out

.PHONY: clean
clean:
	rm packetbeat || true
	rm -r packetbeat-$(VERSION) || true
