
BUILD_DIR=build
COVERAGE_DIR=${BUILD_DIR}/coverage
BEATS=packetbeat filebeat winlogbeat metricbeat
PROJECTS=libbeat ${BEATS}
SNAPSHOT?=yes

# Runs complete testsuites (unit, system, integration) for all beats with coverage and race detection.
# Also it builds the docs and the generators
.PHONY: testsuite
testsuite:
	$(foreach var,$(PROJECTS),$(MAKE) -C $(var) testsuite || exit 1;)
	#$(MAKE) -C generate test

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
	-tail -q -n +2 ./winlogbeat/${COVERAGE_DIR}/*.cov >> ./${COVERAGE_DIR}/full.cov
	-tail -q -n +2 ./libbeat/${COVERAGE_DIR}/*.cov >> ./${COVERAGE_DIR}/full.cov
	go tool cover -html=./${COVERAGE_DIR}/full.cov -o ${COVERAGE_DIR}/full.html

.PHONY: update
update:
	$(foreach var,$(BEATS),$(MAKE) -C $(var) update || exit 1;)

.PHONY: clean
clean:
	rm -rf build
	$(foreach var,$(PROJECTS),$(MAKE) -C $(var) clean || exit 1;)
	$(MAKE) -C generate clean

# Cleans up the vendor directory from unnecessary files
# This should always be run after updating the dependencies
.PHONY: clean-vendor
clean-vendor:
	sh scripts/clean_vendor.sh

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
	mkdir -p build/dashboards
	$(foreach var,$(BEATS),cp -r $(var)/etc/kibana/ build/dashboards/$(var)  || exit 1;)

# Builds the documents for each beat
.PHONY: docs
docs:
	sh libbeat/scripts/build_docs.sh ${PROJECTS}

.PHONY: package
package: beats-dashboards
	$(MAKE) -C libbeat package-setup
	$(foreach var,$(BEATS),SNAPSHOT=$(SNAPSHOT) $(MAKE) -C $(var) package || exit 1;)

	# build the dashboards package
	echo "Start building the dashboards package"
	mkdir -p build/upload/
	BUILD_DIR=${shell pwd}/build SNAPSHOT=$(SNAPSHOT) $(MAKE) -C dev-tools/packer package-dashboards ${shell pwd}/build/upload/build_id.txt
	mv build/upload build/dashboards-upload

	# Copy build files over to top build directory
	mkdir -p build/upload/
	$(foreach var,$(BEATS),cp -r $(var)/build/upload/ build/upload/$(var)  || exit 1;)
	cp -r build/dashboards-upload build/upload/dashboards

# Upload nightly builds to S3
.PHONY: upload-nightlies-s3
upload-nightlies-s3: all
	aws s3 cp --recursive --acl public-read build/upload s3://beats-nightlies

# Run after building to sign packages and publish to APT and YUM repos.
.PHONY: package-upload
upload-package:
	$(MAKE) -C dev-tools/packer deb-rpm-s3
	# You must export AWS_ACCESS_KEY=<AWS access> and export AWS_SECRET_KEY=<secret>
	# before running this make target.
	dev-tools/packer/docker/deb-rpm-s3/deb-rpm-s3.sh

.PHONY: release-upload
upload-release:
	aws s3 cp --recursive --acl public-read build/upload s3://download.elasticsearch.org/beats/
