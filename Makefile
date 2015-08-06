RELEASE?=master
BUILDID?=$(shell date +%y%m%d%H%M%S)

.PHONY: all
all: packetbeat/deb packetbeat/rpm

.PHONY: packetbeat topbeat
packetbeat topbeat: image build
	cd build && xgo -image=tudorg/beats-builder -static \
		-before-build=../xgo-scripts/before_build.sh \
		-branch $(RELEASE) \
		github.com/elastic/$@

%/deb: % build/god-linux-386 build/god-linux-amd64
	ARCH=386 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/debian/build.sh
	ARCH=amd64 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/debian/build.sh

%/rpm: % build/god-linux-386 build/god-linux-amd64
	ARCH=386 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/centos/build.sh
	ARCH=amd64 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/centos/build.sh

.PHONY: deps image
deps:
	go get github.com/tsg/xgo

.PHONY: xgo-image
xgo-image:
	docker build -t tudorg/beats-builder xgo-image/

.PHONY: go-daemon-image
go-daemon-image:
	docker build -t tudorg/go-daemon go-daemon/

build/god-linux-386 build/god-linux-amd64: go-daemon-image
	docker run -v $(shell pwd)/build:/build tudorg/go-daemon

build:
	mkdir -p build


.PHONY: run-interactive
run-interactive:
	docker run -t -i -v $(shell pwd)/build:/build \
		-v $(shell pwd)/xgo-scripts/:/scripts \
		--entrypoint=bash tudorg/beats-builder
.PHONY: clean
clean:
	rm -rf build/ || true
