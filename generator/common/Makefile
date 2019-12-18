PWD=$(shell pwd)
BEAT_TYPE?=beat
BEAT_NAME?=beatpath/test${BEAT_TYPE}
BEAT_PATH=${GOPATH}/src/${BEAT_NAME}
ES_BEATS=${GOPATH}/src/github.com/elastic/beats
PREPARE_COMMAND?=

# Runs test build for mock beat
.PHONY: test
test: prepare-test
	cd ${BEAT_PATH} ; \
	export PATH=$${GOPATH}/bin:$${PATH}; \
	git config user.email "beats-jenkins@test.com" || exit 1 ; \
	git config user.name "beats-jenkins" || exit 1 ; \
	$(MAKE) check CHECK_HEADERS_DISABLED=y || exit 1 ; \
	$(MAKE) || exit 1 ; \
	$(MAKE) unit

.PHONY: prepare-test
prepare-test:: ${GOPATH}/bin/mage
	rm -fr ${BEAT_PATH}
	mkdir -p ${BEAT_PATH}
	cd ${GOPATH}/src/github.com/elastic/beats/ ; \
	export MODULE=elastic ; \
	export METRICSET=test ; \
	export GO111MODULE=off; \
	export NEWBEAT_PROJECT_NAME=Testbeat ; \
	export NEWBEAT_GITHUB_NAME=ruflin ; \
	export NEWBEAT_BEAT_PATH=${BEAT_NAME} ; \
	export NEWBEAT_FULL_NAME="Nicolas Ruflin" ; \
	export NEWBEAT_TYPE=${BEAT_TYPE} ; \
	export NEWBEAT_DEV=1 ; \
	mage GenerateCustomBeat

# Runs test build for the created beat
.PHONY: test-build
test-build: test
	# Copy dev tools
	cp -r ${PWD}/../../../dev-tools ${BEAT_PATH}/vendor/github.com/elastic/beats/

	cd ${BEAT_PATH}/dev-tools/packer ; \
	$(MAKE) deps ; \
	$(MAKE) images

${GOPATH}/bin/mage:
	go get -u -d github.com/magefile/mage
	cd $${GOPATH}/src/github.com/magefile/mage;  \
	go run bootstrap.go

# Cleans up environment
.PHONY: clean
clean:
	rm -rf ${BEAT_PATH}
