RELEASE?=master
DATE:=$(shell date +%y%m%d%H%M%S)
BUILDID?=$(DATE)

.PHONY: all
all: packetbeat/deb packetbeat/rpm packetbeat/darwin packetbeat/win \
	topbeat/deb topbeat/rpm topbeat/darwin topbeat/win


.PHONY: packetbeat topbeat
packetbeat topbeat: xgo-image build
	cd build && xgo -image=tudorg/beats-builder -static \
		-before-build=../xgo-scripts/$@_before_build.sh \
		-branch $(RELEASE) \
		github.com/elastic/$@

%/deb: % build/god-linux-386 build/god-linux-amd64 fpm-image
	ARCH=386 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/debian/build.sh
	ARCH=amd64 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/debian/build.sh

%/rpm: % build/god-linux-386 build/god-linux-amd64 fpm-image
	ARCH=386 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/centos/build.sh
	ARCH=amd64 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/centos/build.sh

%/darwin: % fpm-image
	ARCH=amd64 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/darwin/build.sh

%/win: % fpm-image
	ARCH=amd64 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/windows/build.sh

%/bin: % fpm-image
	ARCH=386 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/binary/build.sh
	ARCH=amd64 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/binary/build.sh

.PHONY: deps xgo-image
deps:
	go get github.com/tsg/xgo
	go get github.com/tsg/gotpl

.PHONY: xgo-image
xgo-image:
	docker build -t tudorg/beats-builder docker/xgo-image/

.PHONY: fpm-image
fpm-image:
	docker build -t tudorg/fpm docker/fpm-image/

.PHONY: go-daemon-image
go-daemon-image:
	docker build -t tudorg/go-daemon docker/go-daemon/

build/god-linux-386 build/god-linux-amd64: go-daemon-image
	docker run -v $(shell pwd)/build:/build tudorg/go-daemon

build:
	mkdir -p build

.PHONY: s3-nightlies-upload
s3-nightlies-upload: all
	echo $(BUILDID) > build/upload/build_id.txt
	aws s3 cp --recursive --acl public-read build/upload s3://beats-nightlies

.PHONY: run-interactive
run-interactive:
	docker run -t -i -v $(shell pwd)/build:/build \
		-v $(shell pwd)/xgo-scripts/:/scripts \
		--entrypoint=bash tudorg/beats-builder
.PHONY: clean
clean:
	rm -rf build/ || true
