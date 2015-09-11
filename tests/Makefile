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
	make -C .. test-binary
	. env/bin/activate; nosetests --processes=${PROCESSES} --process-timeout=$(TIMEOUT)

.PHONY: coverage
coverage: test
	mkdir -p ../coverage
	# Writes count mode on top of file
	echo 'mode: count' > ../coverage/integration.cov
	# Collects all integration coverage files and skipts top line with mode
	tail -q -n +2 ./run/**/*.cov >> ../coverage/integration.cov
	# Generats HTML output
	go tool cover -html=../coverage/integration.cov -o ../coverage/integration.html

.PHONY: clean
clean:
	-rm -r env
	-rm -r run
	-rm last_run
