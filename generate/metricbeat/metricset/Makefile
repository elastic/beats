BUILD_DIR?=build
PWD=$(shell pwd)
PYTHON_ENV=${BUILD_DIR}/python-env/

# Runs test build for mock beat
.PHONY: test
test: python-env
	mkdir -p build/src/beatpath
	cp -r \{\{cookiecutter.beat\}\} build
	cp tests/cookiecutter.json build/
	mkdir -p build/src/github.com/elastic/beats/
	cp -r ../../metricbeat ../../libbeat ../../topbeat ../../vendor build/src/github.com/elastic/beats/
	. ${PYTHON_ENV}/bin/activate; \
	cookiecutter --no-input -o build/src/beatpath -f  build ; \
	cd build/src/beatpath/countbeat ; \
	mkdir -p vendor/github.com/elastic/beats/ ; \
	# Makes sure to use current version of beats ; \
	cp -r ${PWD}/../../metricbeat ${PWD}/../../libbeat ${PWD}/../../topbeat ${PWD}/../../vendor vendor/github.com/elastic/beats/ ; \
	export GOPATH=${PWD}/build ; \
	export PATH=${PATH}:${PWD}/build/bin; \
	make init ; \
	make check ; \
	make
	#make testsuite ; \
	#cd dev-tools/packer ; \
	#make deps ; \
	#make images ; \
	#make

# Sets up the virtual python environment
.PHONY: python-env
python-env:
	test -d ${PYTHON_ENV} || virtualenv ${PYTHON_ENV}
	. ${PYTHON_ENV}/bin/activate && pip install --upgrade pip cookiecutter PyYAML

# Cleans up environment
.PHONY: clean
clean:
	rm -rf build
