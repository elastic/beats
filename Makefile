BUILD_DIR=build
COVERAGE_DIR=${BUILD_DIR}/coverage
PROJECTS=packetbeat topbeat filebeat winlogbeat

# Runs testsuites for all beats
testsuite:
	$(foreach var,$(PROJECTS),make -C $(var) testsuite;)

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
	$(foreach var,$(PROJECTS),make -C $(var) clean;)

check:
	$(foreach var,$(PROJECTS),make -C $(var) check;)

fmt:
	$(foreach var,$(PROJECTS),make -C $(var) fmt;)

simplify:
	$(foreach var,$(PROJECTS),make -C $(var) simplify;)
