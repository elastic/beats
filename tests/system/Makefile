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
	echo 'mode: count' > ../coverage/system.cov
	# Collects all system coverage files and skips top line with mode
	tail -q -n +2 ./run/**/*.cov >> ../coverage/system.cov
	# Generates HTML output
	go tool cover -html=../coverage/system.cov -o ../coverage/system.html

.PHONY: clean
clean:
	-rm -r run
	-rm last_run
