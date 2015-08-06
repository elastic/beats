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

%/deb: %
	ARCH=386 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/debian/build.sh
	ARCH=amd64 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/debian/build.sh

%/rpm: %
	ARCH=386 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/centos/build.sh
	ARCH=amd64 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/centos/build.sh

.PHONY: deps image
deps:
	go get github.com/tsg/xgo

.PHONY: image
image:
	docker build -t tudorg/beats-builder xgo-image/

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
