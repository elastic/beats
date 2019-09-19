BUILD_DIR?=build
PWD=$(shell pwd)
PYTHON_ENV?=${BUILD_DIR}/python-env
BEAT_TYPE?=beat
BEAT_PATH=${BUILD_DIR}/src/beatpath/testbeat
ES_BEATS=${GOPATH}/src/github.com/elastic/beats
PREPARE_COMMAND?=

# Runs test build for mock beat
.PHONY: test
test: prepare-test
	. ${PYTHON_ENV}/bin/activate; \
	export GOPATH=${PWD}/build ; \
	export PATH=$${GOPATH}/bin:${PATH}; \
	cd ${BEAT_PATH} ; \
	${PREPARE_COMMAND} \
	$(MAKE) update fmt || exit 1 ; \
	git config user.email "beats-jenkins@test.com" || exit 1 ; \
	git config user.name "beats-jenkins" || exit 1 ; \
	$(MAKE) check CHECK_HEADERS_DISABLED=y || exit 1 ; \
	$(MAKE) || exit 1 ; \
	$(MAKE) unit

.PHONY: prepare-test
prepare-test:: python-env
	# Makes sure to use current version of beats for testing
	rm -fr ${BUILD_DIR}/src/github.com/elastic/beats/
	git clone -s ${PWD}/../../ ${BUILD_DIR}/src/github.com/elastic/beats/
	rm -fr ${BEAT_PATH}
	mkdir -p ${BEAT_PATH}
	cd ${PWD}/${BUILD_DIR}/src/github.com/elastic/beats/ ; \
	export MODULE=elastic ; \
	export METRICSET=test ; \
	export GO111MODULE=off; \
	export GOPATH=${PWD}/build ; \
	export NEWBEAT_PROJECT_NAME=Testbeat ; \
	export NEWBEAT_GITHUB_NAME=ruflin ; \
	export NEWBEAT_BEAT_PATH=beatpath/testbeat ; \
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

# Sets up the virtual python environment
.PHONY: python-env
python-env:
	@test -d ${PYTHON_ENV} || virtualenv ${PYTHON_ENV}
	@${PYTHON_ENV}/bin/pip install --upgrade pip PyYAML
	@# Work around pip bug. See: https://github.com/pypa/pip/issues/4464
	@find $(PYTHON_ENV) -type d -name dist-packages -exec sh -c "echo dist-packages > {}.pth" ';'

# Cleans up environment
.PHONY: clean
clean:
	@rm -rf build
