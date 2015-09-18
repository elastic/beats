SHELL := /bin/bash
PROCESSES ?= 4
TIMEOUT ?= 30

.PHONY: test
test:
	make -C .. packetbeat.test
	make -C .. env
	. ../env/bin/activate && nosetests --processes=${PROCESSES} --process-timeout=$(TIMEOUT)

.PHONY: coverage
coverage: test
	mkdir -p ../coverage
	# Writes count mode on top of file
	echo 'mode: count' > ../coverage/integration.cov
	# Collects all integration coverage files and skips top line with mode
	tail -q -n +2 ./run/**/*.cov >> ../coverage/integration.cov
	# Generates HTML output
	go tool cover -html=../coverage/integration.cov -o ../coverage/integration.html

.PHONY: clean
clean:
	-rm -r run
	-rm last_run
