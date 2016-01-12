DATE:=$(shell date +%y%m%d%H%M%S)
BUILDID?=$(DATE)


.PHONY: all
all: packetbeat/deb packetbeat/rpm packetbeat/darwin packetbeat/win packetbeat/bin \
	topbeat/deb topbeat/rpm topbeat/darwin topbeat/win topbeat/bin \
	filebeat/deb filebeat/rpm filebeat/darwin filebeat/win filebeat/bin \
	winlogbeat/win \
	build/upload/build_id.txt latest

.PHONY: packetbeat topbeat filebeat
packetbeat topbeat filebeat winlogbeat: build/upload
	# cross compile on ubuntu
	cd build && xgo -image=tudorg/beats-builder \
		-before-build=../xgo-scripts/$@_before_build.sh \
		-pkg $@ \
		-source ../../.. \
		github.com/elastic/beats
	# linux builds on debian 6
	cd build && xgo -image=tudorg/beats-builder-deb6 \
		-before-build=../xgo-scripts/$@_before_build.sh \
		-pkg $@ \
		-source ../../.. \
		github.com/elastic/beats

%/deb: % build/god-linux-386 build/god-linux-amd64 fpm-image
	ARCH=386 BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/debian/build.sh
	ARCH=amd64 BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/debian/build.sh

%/rpm: % build/god-linux-386 build/god-linux-amd64 fpm-image
	ARCH=386 BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/centos/build.sh
	ARCH=amd64 BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/centos/build.sh

%/darwin: %
	ARCH=amd64 BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/darwin/build.sh

%/win: %
	ARCH=amd64 BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/windows/build.sh

%/bin: %
	ARCH=386 BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/binary/build.sh
	ARCH=amd64 BEAT=$(@D) BUILDID=$(BUILDID) ./platforms/binary/build.sh

.PHONY: deps
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

build/god-linux-386 build/god-linux-amd64:
	docker run -v $(shell pwd)/build:/build tudorg/go-daemon

build/upload:
	mkdir -p build/upload

build/upload/build_id.txt:
	echo $(BUILDID) > build/upload/build_id.txt

.PHONY: s3-nightlies-upload
s3-nightlies-upload: all
	aws s3 cp --recursive --acl public-read build/upload s3://beats-nightlies

# Build the image required for package-upload.
.PHONY: deb-rpm-s3
deb-rpm-s3:
	docker/deb-rpm-s3/build.sh

# Run after building to sign packages and publish to APT and YUM repos.
.PHONY: package-upload
package-upload:
	# You must export AWS_ACCESS_KEY=<AWS access> and export AWS_SECRET_KEY=<secret>
	# before running this make target.
	docker/deb-rpm-s3/deb-rpm-s3.sh

.PHONY: release-upload
release-upload:
	aws s3 cp --recursive --acl public-read build/upload s3://download.elasticsearch.org/beats/

.PHONY: run-interactive
run-interactive:
	docker run -t -i -v $(shell pwd)/build:/build \
		-v $(shell pwd)/xgo-scripts/:/scripts \
		--entrypoint=bash tudorg/beats-builder-deb6

.PHONY: images
images: xgo-image fpm-image go-daemon-image

.PHONY: push-images
push-images:
	docker push tudorg/beats-builder
	docker push tudorg/beats-builder-deb6
	docker push tudorg/fpm
	docker push tudorg/go-daemon

.PHONY: pull-images
pull-images:
	docker pull tudorg/beats-builder
	docker pull tudorg/beats-builder-deb6
	docker pull tudorg/fpm
	docker pull tudorg/go-daemon

.PHONY: clean
clean:
	rm -rf build/ || true
	-docker rm -v build-image

# Creates a latest file for the most recent build
.PHONY: latest
latest:
	BUILDID=${BUILDID} \
	./xgo-scripts/latest.sh
