BUILD_DIR?=build
PWD=$(shell pwd)
PYTHON_ENV?=${BUILD_DIR}/python-env
BEAT_PATH=${BUILD_DIR}/src/beatpath/testbeat



# Runs test build for mock beat
.PHONY: test
test: python-env

	# Makes sure to use current version of beats for testing
	mkdir -p ${BUILD_DIR}/src/github.com/elastic/beats/
	rsync -a --exclude=build ${PWD}/../../* ${BUILD_DIR}/src/github.com/elastic/beats/

	mkdir -p ${BEAT_PATH}
	export GOPATH=${PWD}/build ; \
	. ${PYTHON_ENV}/bin/activate && python ${PWD}/build/src/github.com/elastic/beats/generate/metricbeat.py --project_name=Testbeat --github_name=ruflin --beat_path=beatpath/testbeat --full_name="Nicolas Ruflin"

	. ${PYTHON_ENV}/bin/activate; \
	export GOPATH=${PWD}/build ; \
	export PATH=${PATH}:${PWD}/build/bin; \
	cd ${BEAT_PATH} ; \
	make copy-vendor ; \
	MODULE=elastic METRICSET=test make create-metricset ; \
	make check ; \
	make update ; \
	make ; \
	make unit


# Tests the build process for the beat
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
	test -d ${PYTHON_ENV} || virtualenv ${PYTHON_ENV}
	. ${PYTHON_ENV}/bin/activate && pip install --upgrade pip cookiecutter PyYAML

# Cleans up environment
.PHONY: clean
clean:
	rm -rf build
