RELEASE?=master
DATE:=$(shell date +%y%m%d%H%M%S)
BUILDID?=$(DATE)

.PHONY: all
all: packetbeat/deb packetbeat/rpm packetbeat/darwin packetbeat/win packetbeat/bin \
	topbeat/deb topbeat/rpm topbeat/darwin topbeat/win topbeat/bin


.PHONY: packetbeat topbeat
packetbeat topbeat: xgo-image build
	cd build && xgo -image=tudorg/beats-builder \
		-before-build=../xgo-scripts/$@_before_build.sh \
		-branch $(RELEASE) \
		github.com/elastic/$@

.PHONY: packetbeat-linux topbeat-linux
packetbeat-linux: xgo-image build
	cd build && xgo -image=tudorg/beats-builder-deb6 \
		-before-build=../xgo-scripts/packetbeat_before_build.sh \
		-branch $(RELEASE) \
		github.com/elastic/packetbeat
topbeat-linux: xgo-image build
	cd build && xgo -image=tudorg/beats-builder-deb6 \
		-before-build=../xgo-scripts/topbeat_before_build.sh \
		-branch $(RELEASE) \
		github.com/elastic/topbeat

%/deb: %-linux build/god-linux-386 build/god-linux-amd64 fpm-image
	ARCH=386 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/debian/build.sh
	ARCH=amd64 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/debian/build.sh

%/rpm: %-linux build/god-linux-386 build/god-linux-amd64 fpm-image
	ARCH=386 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/centos/build.sh
	ARCH=amd64 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/centos/build.sh

%/darwin: % fpm-image
	ARCH=amd64 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/darwin/build.sh

%/win: % fpm-image
	ARCH=amd64 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/windows/build.sh

%/bin: %-linux fpm-image
	ARCH=386 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/binary/build.sh
	ARCH=amd64 RELEASE=$(RELEASE) BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/binary/build.sh

.PHONY: deps xgo-image
deps:
	go get github.com/tsg/xgo
	go get github.com/tsg/gotpl

.PHONY: xgo-image
xgo-image:
	cd docker/xgo-image/; ./build.sh
	cd docker/xgo-image-deb6/; ./build.sh

.PHONY: fpm-image
fpm-image:
	docker build --rm=true -t tudorg/fpm docker/fpm-image/

.PHONY: go-daemon-image
go-daemon-image:
	docker build --rm=true -t tudorg/go-daemon docker/go-daemon/

build/god-linux-386 build/god-linux-amd64: go-daemon-image
	docker run -v $(shell pwd)/build:/build tudorg/go-daemon

build:
	mkdir -p build

.PHONY: s3-nightlies-upload
s3-nightlies-upload: all
	echo $(BUILDID) > build/upload/build_id.txt
	aws s3 cp --recursive --acl public-read build/upload s3://beats-nightlies

.PHONY: release-upload
release-upload:
	aws s3 cp --recursive --acl public-read build/upload s3://download.elasticsearch.org/beats/

.PHONY: run-interactive
run-interactive:
	docker run -t -i -v $(shell pwd)/build:/build \
		-v $(shell pwd)/xgo-scripts/:/scripts \
		--entrypoint=bash tudorg/beats-builder-deb6
.PHONY: clean
clean:
	rm -rf build/ || true
