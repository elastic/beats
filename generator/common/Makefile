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
	$(MAKE) copy-vendor || exit 1  ; \
	${PREPARE_COMMAND} \
	$(MAKE) git-init || exit 1 ; \
	$(MAKE) update fmt || exit 1 ; \
	git config user.email "beats-jenkins@test.com" || exit 1 ; \
	git config user.name "beats-jenkins" || exit 1 ; \
	$(MAKE) git-add || exit 1 ; \
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
	export GOPATH=${PWD}/build ; \
	. ${PYTHON_ENV}/bin/activate && \
	  python ${PWD}/build/src/github.com/elastic/beats/script/generate.py \
	    --type=${BEAT_TYPE} \
	    --project_name=Testbeat \
	    --github_name=ruflin \
	    --beat_path=beatpath/testbeat \
	    --full_name="Nicolas Ruflin"

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
