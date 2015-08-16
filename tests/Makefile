SHELL := /bin/bash

env: env/bin/activate
env/bin/activate: requirements.txt
	test -d env || virtualenv env > /dev/null
	. env/bin/activate && pip install -Ur requirements.txt > /dev/null
	touch env/bin/activate

.PHONY: test
test: env
	make -C ..
	. env/bin/activate; nosetests --processes=4

clean:
	-rm last_run
	-rm -r run
	-rm -r env