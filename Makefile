BUILD_DIR=build
COVERAGE_DIR=${BUILD_DIR}/coverage
BEATS=packetbeat topbeat filebeat winlogbeat
PROJECTS=libbeat ${BEATS}

# Runs testsuites for all beats
testsuite:
	$(foreach var,$(PROJECTS),make -C $(var) testsuite || exit 1;)

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

update:
	$(foreach var,$(BEATS),make -C $(var) update || exit 1;)

clean:
	$(foreach var,$(PROJECTS),make -C $(var) clean || exit 1;)

check:
	$(foreach var,$(PROJECTS),make -C $(var) check || exit 1;)

fmt:
	$(foreach var,$(PROJECTS),make -C $(var) fmt || exit 1;)

simplify:
	$(foreach var,$(PROJECTS),make -C $(var) simplify || exit 1;)
