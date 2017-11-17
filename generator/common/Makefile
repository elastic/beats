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
	export PATH=${PATH}:${PWD}/build/bin; \
	cd ${BEAT_PATH} ; \
	make copy-vendor ; \
	${PREPARE_COMMAND}  \
	make check ; \
	make update ; \
	make ; \
	make unit

prepare-test: python-env
	# Makes sure to use current version of beats for testing
	mkdir -p ${BUILD_DIR}/src/github.com/elastic/beats/
	rsync -a --exclude=build ${PWD}/../../* ${BUILD_DIR}/src/github.com/elastic/beats/

	mkdir -p ${BEAT_PATH}
	export GOPATH=${PWD}/build ; \
	. ${PYTHON_ENV}/bin/activate && python ${PWD}/build/src/github.com/elastic/beats/script/generate.py --type=${BEAT_TYPE} --project_name=Testbeat --github_name=ruflin --beat_path=beatpath/testbeat --full_name="Nicolas Ruflin"

# Runs test build for the created beat
.PHONY: test-build
test-build: test
	# Copy dev tools
	cp -r ${PWD}/../../../dev-tools ${BEAT_PATH}/vendor/github.com/elastic/beats/

	cd ${BEAT_PATH}/dev-tools/packer ; \
	make deps ; \
	make images

# Sets up the virtual python environment
.PHONY: python-env
python-env:
	@test -d ${PYTHON_ENV} || virtualenv ${PYTHON_ENV}
	@${PYTHON_ENV}/bin/pip install --upgrade pip PyYAML

# Cleans up environment
.PHONY: clean
clean:
	@rm -rf build
