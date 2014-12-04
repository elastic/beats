SHELL := /bin/bash

env: env/bin/activate
env/bin/activate: requirements.txt
	test -d env || virtualenv env
	. env/bin/activate; pip install -Ur requirements.txt
	touch env/bin/activate

.PHONY: test
test: env
	make -C ..
	. env/bin/activate; nosetests
