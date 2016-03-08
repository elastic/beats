BUILD_DIR=build
COVERAGE_DIR=${BUILD_DIR}/coverage
BEATS=packetbeat topbeat filebeat winlogbeat metricbeat
PROJECTS=libbeat ${BEATS}

# Runs complete testsuites (unit, system, integration) for all beats,
# with coverage and race detection.
.PHONY: testsuite
testsuite:
	$(foreach var,$(PROJECTS),$(MAKE) -C $(var) testsuite || exit 1;)

# Runs unit and system tests without coverage and race detection.
.PHONY: test
test:
	$(foreach var,$(PROJECTS),$(MAKE) -C $(var) test || exit 1;)

# Runs unit tests without coverage and race detection.
.PHONY: unit
unit:
	$(foreach var,$(PROJECTS),$(MAKE) -C $(var) unit || exit 1;)

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

.PHONY: update
update:
	$(foreach var,$(BEATS),$(MAKE) -C $(var) update || exit 1;)

.PHONY: clean
clean:
	$(foreach var,$(PROJECTS),$(MAKE) -C $(var) clean || exit 1;)

.PHONY: check
check:
	$(foreach var,$(PROJECTS),$(MAKE) -C $(var) check || exit 1;)

.PHONY: fmt
fmt:
	$(foreach var,$(PROJECTS),$(MAKE) -C $(var) fmt || exit 1;)

.PHONY: simplify
simplify:
	$(foreach var,$(PROJECTS),$(MAKE) -C $(var) simplify || exit 1;)

# Collects all dashboards and generates dashboard folder for https://github.com/elastic/beats-dashboards/tree/master/dashboards
.PHONY: beats-dashboards
beats-dashboards:
	mkdir -p build
	$(foreach var,$(PROJECTS),cp -r $(var)/etc/kibana/ build/dashboards  || exit 1;)

