BUILDID?=$(shell git rev-parse HEAD)
SNAPSHOT?=yes

BEATS_BUILDER_IMAGE?=tudorg/beats-builder
BEATS_BUILDER_DEB_IMAGE?=tudorg/beats-builder-deb7
BEATS_GOPATH=$(firstword $(subst :, ,${GOPATH}))

makefile_abspath:=$(abspath $(lastword $(MAKEFILE_LIST)))
packer_absdir=$(shell dirname ${makefile_abspath})
beat_abspath=${BEATS_GOPATH}/src/${BEAT_PATH}


%/deb: ${BUILD_DIR}/god-linux-386 ${BUILD_DIR}/god-linux-amd64 fpm-image
	echo Creating DEB packages for $(@D)
	ARCH=386 BEAT=$(@D) BUILD_DIR=${BUILD_DIR} UPLOAD_DIR=${UPLOAD_DIR} BEAT_PATH=$(beat_abspath) BUILDID=$(BUILDID) SNAPSHOT=$(SNAPSHOT) $(packer_absdir)/platforms/debian/build.sh
	ARCH=amd64 BEAT=$(@D) BUILD_DIR=${BUILD_DIR} UPLOAD_DIR=${UPLOAD_DIR} BEAT_PATH=$(beat_abspath) BUILDID=$(BUILDID) SNAPSHOT=$(SNAPSHOT) $(packer_absdir)/platforms/debian/build.sh

%/rpm: ${BUILD_DIR}/god-linux-386 ${BUILD_DIR}/god-linux-amd64 fpm-image
	echo Creating RPM packages for $(@D)
	ARCH=386 BEAT=$(@D) BUILD_DIR=${BUILD_DIR} UPLOAD_DIR=${UPLOAD_DIR} BEAT_PATH=$(beat_abspath) BUILDID=$(BUILDID) SNAPSHOT=$(SNAPSHOT) $(packer_absdir)/platforms/centos/build.sh
	ARCH=amd64 BEAT=$(@D) BUILD_DIR=${BUILD_DIR} UPLOAD_DIR=${UPLOAD_DIR} BEAT_PATH=$(beat_abspath) BUILDID=$(BUILDID) SNAPSHOT=$(SNAPSHOT) $(packer_absdir)/platforms/centos/build.sh

%/darwin:
	echo Creating Darwin packages for $(@D)
	ARCH=amd64 BEAT=$(@D) BUILD_DIR=${BUILD_DIR} UPLOAD_DIR=${UPLOAD_DIR} BEAT_PATH=$(beat_abspath) BUILDID=$(BUILDID) SNAPSHOT=$(SNAPSHOT) $(packer_absdir)/platforms/darwin/build.sh

%/win:
	echo Creating Darwin packages for $(@D)
	ARCH=386 BEAT=$(@D) BUILD_DIR=${BUILD_DIR} UPLOAD_DIR=${UPLOAD_DIR} BEAT_PATH=$(beat_abspath) BUILDID=$(BUILDID) SNAPSHOT=$(SNAPSHOT) $(packer_absdir)/platforms/windows/build.sh
	ARCH=amd64 BEAT=$(@D) BUILD_DIR=${BUILD_DIR} UPLOAD_DIR=${UPLOAD_DIR} BEAT_PATH=$(beat_abspath) BUILDID=$(BUILDID) SNAPSHOT=$(SNAPSHOT) $(packer_absdir)/platforms/windows/build.sh

%/bin:
	echo Creating Linux packages for $(@D)
	ARCH=386 BEAT=$(@D) BUILD_DIR=${BUILD_DIR} UPLOAD_DIR=${UPLOAD_DIR} BEAT_PATH=$(beat_abspath) BUILDID=$(BUILDID) SNAPSHOT=$(SNAPSHOT) $(packer_absdir)/platforms/binary/build.sh
	ARCH=amd64 BEAT=$(@D) BUILD_DIR=${BUILD_DIR} UPLOAD_DIR=${UPLOAD_DIR} BEAT_PATH=$(beat_abspath) BUILDID=$(BUILDID) SNAPSHOT=$(SNAPSHOT) $(packer_absdir)/platforms/binary/build.sh

.PHONY: package-dashboards
package-dashboards:
	echo Creating the Dashboards package
	UPLOAD_DIR=${UPLOAD_DIR} BUILDID=$(BUILDID) SNAPSHOT=$(SNAPSHOT) $(packer_absdir)/platforms/dashboards/build.sh

.PHONY: deps
deps:
	cd ../vendor/github.com/tsg/gotpl; go install

.PHONY: xgo-image
xgo-image:
	cd $(packer_absdir)/docker/xgo-image/; ./build.sh
	# copy build.sh script in the xgo-image-deb7 to avoid code duplication
	cp $(packer_absdir)/docker/xgo-image/base/build.sh $(packer_absdir)/docker/xgo-image-deb7/base/build.sh
	cd $(packer_absdir)/docker/xgo-image-deb7/; ./build.sh

.PHONY: fpm-image
fpm-image:
	docker build --rm=true -t tudorg/fpm $(packer_absdir)/docker/fpm-image

.PHONY: go-daemon-image
go-daemon-image:
	docker build --rm=true -t tudorg/go-daemon $(packer_absdir)/docker/go-daemon/

${BUILD_DIR}/god-linux-386 ${BUILD_DIR}/god-linux-amd64:
	docker run --rm -v ${BUILD_DIR}:/build tudorg/go-daemon

${UPLOAD_DIR}:
	mkdir -p ${UPLOAD_DIR}

${UPLOAD_DIR}/build_id.txt:
	echo $(BUILDID) > ${UPLOAD_DIR}/build_id.txt

# Build the image required for package-upload.
.PHONY: deb-rpm-s3
deb-rpm-s3:
	$(packer_absdir)/docker/deb-rpm-s3/build.sh

.PHONY: run-interactive-builder-deb
run-interactive-builder-deb:
	docker run -t -i -v $(shell pwd)/build:/build \
		-v $(shell pwd)/xgo-scripts/:/scripts \
		-v $(shell pwd)/../..:/source \
		--entrypoint=bash ${BEATS_BUILDER_DEB_IMAGE}

.PHONY: run-interactive-builder
run-interactive-builder:
	docker run -t -i -v $(shell pwd)/build:/build \
		-v $(packer_absdir)/xgo-scripts/:/scripts \
		-v $(shell pwd)/../..:/source \
		--entrypoint=bash ${BEATS_BUILDER_IMAGE}

.PHONY: images
images: xgo-image fpm-image go-daemon-image

.PHONY: push-images
push-images:
	docker push ${BEATS_BUILDER_IMAGE}
	docker push ${BEATS_BUILDER_DEB_IMAGE}
	docker push tudorg/fpm
	docker push tudorg/go-daemon

.PHONY: pull-images
pull-images:
	docker pull ${BEATS_BUILDER_IMAGE}
	docker pull ${BEATS_BUILDER_DEB_IMAGE}
	docker pull tudorg/fpm
	docker pull tudorg/go-daemon


define rm-image =
@echo "Cleaning $(1) image..."
  @if [ $(shell docker ps  -n 1 -a -q  --filter="image=$(1)" )  ]; then \
      docker stop $(shell docker ps -a -q --filter="image=$(1)"); \
      docker rm   $(shell docker ps -a -q --filter="image=$(1)"); \
  fi; \
\
  if [ $(shell docker images -q $(1)) ]; then \
    docker rmi $(1); \
  fi
endef


.PHONY: clean-images
clean-images:
	@$(call rm-image, ${BEATS_BUILDER_DEB_IMAGE})
	@$(call rm-image, ${BEATS_BUILDER_IMAGE})

.PHONY: clean
clean:
	$(call rm-image,build-image)
