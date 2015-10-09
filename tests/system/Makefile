

env: env/bin/activate
env/bin/activate: requirements.txt
	test -d env || virtualenv env > /dev/null
	. env/bin/activate && pip install -Ur requirements.txt > /dev/null
	touch env/bin/activate

.PHONY: system-tests
system-tests: env
	make -C ../.. libbeat.test
	make -C ../.. prepare-tests
	. env/bin/activate; nosetests --processes=${PROCESSES} --process-timeout=$(TIMEOUT)
	# Writes count mode on top of file
	echo 'mode: count' > ${COVERAGE_DIR}/system.cov
	# Collects all system coverage files and skips top line with mode
	tail -q -n +2 ./run/**/*.cov >> ${COVERAGE_DIR}/system.cov

.PHONY: clean
clean:
	-rm -r env
	-rm -r run
	-rm last_run
