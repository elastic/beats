BUILD_DIR?=build
PWD=$(shell pwd)
PYTHON_ENV?=${BUILD_DIR}/python-env
BEAT_PATH=${BUILD_DIR}/src/beatpath/countbeat


# Runs test build for the count metricset
.PHONY: test
test: python-env

	# Create copy of cookiecutter for building with defaults
	mkdir -p build/
	cp -r \{\{cookiecutter.beat\}\} build
	cp tests/cookiecutter.json build/

	mkdir -p ${BEAT_PATH}
	. ${PYTHON_ENV}/bin/activate && cookiecutter --no-input -o build/src/beatpath -f build

	# Makes sure to use current version of beats for testing
	mkdir -p ${BUILD_DIR}/src/github.com/elastic/beats/
	rsync -a --exclude=generate ${PWD}/../../../* ${BUILD_DIR}/src/github.com/elastic/beats/

	. ${PYTHON_ENV}/bin/activate; \
	export GOPATH=${PWD}/build ; \
	export PATH=${PATH}:${PWD}/build/bin; \
	cd ${BEAT_PATH}; \
	make copy-vendor; \
	MODULE=elastic METRICSET=test make create-metricset ; \
	make imports collect check ; \
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
