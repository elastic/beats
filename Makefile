BUILD_DIR?=build
PWD=$(shell pwd)
PYTHON_ENV=${BUILD_DIR}/python-env/

.PHONY: test
test: python-env
	mkdir -p build/src/beatpath
	cp -r \{\{cookiecutter.beat\}\} build
	cp tests/cookiecutter.json build/
	. build/python-env/bin/activate; cookiecutter --no-input -o build/src/beatpath -f  build

	cd build/src/beatpath/testbeat; \
	export GOPATH=${PWD}/build; \
	export GO15VENDOREXPERIMENT=1; \
	make init ; \
	make check ; \
	make

# Sets up the virtual python environment
.PHONY: python-env
python-env:
	test -d ${PYTHON_ENV} || virtualenv ${PYTHON_ENV}
	. ${PYTHON_ENV}/bin/activate && pip install cookiecutter PyYAML

.PHONY: clean
clean:
	rm -rf build
