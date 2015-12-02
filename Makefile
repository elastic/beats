BUILD_DIR=build
COVERAGE_DIR=${BUILD_DIR}/coverage

# Runs testsuites for all beats
testsuite:
	make -C filebeat testsuite
	make -C packetbeat testsuite
	make -C topbeat testsuite
	make -C winlogbeat testsuite
	make -C libbeat testsuite

.PHONY: coverage-report
coverage-report:
	mkdir -p ${COVERAGE_DIR}
	# Writes atomic mode on top of file
	echo 'mode: atomic' > ./${COVERAGE_DIR}/full.cov
	# Collects all coverage files and skips top line with mode
	-tail -q -n +2 ./filebeat/${COVERAGE_DIR}/*.cov >> ./${COVERAGE_DIR}/full.cov
	-tail -q -n +2 ./packetbeat/${COVERAGE_DIR}/*.cov >> ./${COVERAGE_DIR}/full.cov
	-tail -q -n +2 ./topbeat/${COVERAGE_DIR}/*.cov >> ./${COVERAGE_DIR}/full.cov
	-tail -q -n +2 ./winlogbeat/${COVERAGE_DIR}/*.cov >> ./${COVERAGE_DIR}/full.cov
	-tail -q -n +2 ./libbeat/${COVERAGE_DIR}/*.cov >> ./${COVERAGE_DIR}/full.cov
	go tool cover -html=./${COVERAGE_DIR}/full.cov -o ${COVERAGE_DIR}/full.html


clean:
	make -C filebeat clean
	make -C packetbeat clean
	make -C topbeat clean
	make -C winlogbeat clean
	make -C libbeat clean

check:
	make -C filebeat check
	make -C packetbeat check
	make -C topbeat check
	make -C winlogbeat check
	make -C libbeat check
