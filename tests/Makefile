SHELL := /bin/bash
PROCESSES ?= 4
TIMEOUT ?= 30

env: env/bin/activate
env/bin/activate: requirements.txt
	test -d env || virtualenv env > /dev/null
	. env/bin/activate && pip install -Ur requirements.txt > /dev/null
	touch env/bin/activate

.PHONY: test
test: env
	make -C ..
	. env/bin/activate; nosetests --processes=${PROCESSES} --process-timeout=$(TIMEOUT)
