BUILD_DIR=$(CURDIR)/build
COVERAGE_DIR=$(BUILD_DIR)/coverage
BEATS=packetbeat filebeat winlogbeat metricbeat heartbeat auditbeat
PROJECTS=libbeat $(BEATS)
PROJECTS_ENV=libbeat filebeat metricbeat
SNAPSHOT?=yes
PYTHON_ENV?=$(BUILD_DIR)/python-env
VIRTUALENV_PARAMS?=
FIND=find . -type f -not -path "*/vendor/*" -not -path "*/build/*" -not -path "*/.git/*"
GOLINT=golint
GOLINT_REPO=github.com/golang/lint/golint
REVIEWDOG=reviewdog
REVIEWDOG_OPTIONS?=-diff "git diff master"
REVIEWDOG_REPO=github.com/haya14busa/reviewdog/cmd/reviewdog

# Runs complete testsuites (unit, system, integration) for all beats with coverage and race detection.
# Also it builds the docs and the generators

.PHONY: testsuite
testsuite:
	@$(foreach var,$(PROJECTS),$(MAKE) -C $(var) testsuite || exit 1;)

.PHONY: setup-commit-hook
setup-commit-hook:
	@cp script/pre_commit.sh .git/hooks/pre-commit
	@chmod 751 .git/hooks/pre-commit

stop-environments:
	@$(foreach var,$(PROJECTS_ENV),$(MAKE) -C $(var) stop-environment || exit 0;)

# Runs unit and system tests without coverage and race detection.
.PHONY: test
test:
	@$(foreach var,$(PROJECTS),$(MAKE) -C $(var) test || exit 1;)

# Runs unit tests without coverage and race detection.
.PHONY: unit
unit:
	@$(foreach var,$(PROJECTS),$(MAKE) -C $(var) unit || exit 1;)

# Crosscompile all beats.
.PHONY: crosscompile
crosscompile:
	@$(foreach var,filebeat winlogbeat metricbeat heartbeat auditbeat,$(MAKE) -C $(var) crosscompile || exit 1;)

.PHONY: coverage-report
coverage-report:
	@mkdir -p $(COVERAGE_DIR)
	@echo 'mode: atomic' > ./$(COVERAGE_DIR)/full.cov
	@# Collects all coverage files and skips top line with mode
	@$(foreach var,$(PROJECTS),tail -q -n +2 ./$(var)/$(COVERAGE_DIR)/*.cov >> ./$(COVERAGE_DIR)/full.cov || true;)
	@go tool cover -html=./$(COVERAGE_DIR)/full.cov -o $(COVERAGE_DIR)/full.html
	@echo "Generated coverage report $(COVERAGE_DIR)/full.html"

.PHONY: update
update: notice
	@$(foreach var,$(PROJECTS),$(MAKE) -C $(var) update || exit 1;)
	@$(MAKE) -C deploy/kubernetes all

.PHONY: clean
clean:
	@rm -rf build
	@$(foreach var,$(PROJECTS),$(MAKE) -C $(var) clean || exit 1;)
	@$(MAKE) -C generator clean

# Cleans up the vendor directory from unnecessary files
# This should always be run after updating the dependencies
.PHONY: clean-vendor
clean-vendor:
	@sh script/clean_vendor.sh

.PHONY: check
check: python-env
	@$(foreach var,$(PROJECTS),$(MAKE) -C $(var) check || exit 1;)
	@# Checks also python files which are not part of the beats
	@$(FIND) -name *.py -exec $(PYTHON_ENV)/bin/autopep8 -d --max-line-length 120  {} \; | (! grep . -q) || (echo "Code differs from autopep8's style" && false)
	@# Validate that all updates were committed
	@$(MAKE) update
	@git diff | cat
	@git update-index --refresh
	@git diff-index --exit-code HEAD --

# Corrects spelling errors
.PHONY: misspell
misspell:
	go get github.com/client9/misspell
	# Ignore Kibana files (.json)
	$(FIND) -not -path "*.json" -name '*' -exec misspell -w {} \;

.PHONY: fmt
fmt: python-env
	@$(foreach var,$(PROJECTS),$(MAKE) -C $(var) fmt || exit 1;)
	@# Cleans also python files which are not part of the beats
	@$(FIND) -name "*.py" -exec $(PYTHON_ENV)/bin/autopep8 --in-place --max-line-length 120 {} \;

.PHONY: lint
lint:
	@go get $(GOLINT_REPO) $(REVIEWDOG_REPO)
	$(REVIEWDOG) $(REVIEWDOG_OPTIONS)

# Collects all dashboards and generates dashboard folder for https://github.com/elastic/beats-dashboards/tree/master/dashboards
.PHONY: beats-dashboards
beats-dashboards:
	@mkdir -p build/dashboards
	@$(foreach var,$(BEATS),cp -r $(var)/_meta/kibana/ build/dashboards/$(var)  || exit 1;)

# Builds the documents for each beat
.PHONY: docs
docs:
	@$(foreach var,$(PROJECTS),BUILD_DIR=${BUILD_DIR} $(MAKE) -C $(var) docs || exit 1;)
	sh ./script/build_docs.sh dev-guide github.com/elastic/beats/docs/devguide ${BUILD_DIR}

.PHONY: package-all
package-all: update beats-dashboards
	@$(foreach var,$(BEATS),SNAPSHOT=$(SNAPSHOT) $(MAKE) -C $(var) package-all || exit 1;)

	@echo "Start building the dashboards package"
	@mkdir -p build/upload/
	@BUILD_DIR=${BUILD_DIR} UPLOAD_DIR=${BUILD_DIR}/upload SNAPSHOT=$(SNAPSHOT) $(MAKE) -C dev-tools/packer package-dashboards ${BUILD_DIR}/upload/build_id.txt
	@mv build/upload build/dashboards-upload

	@# Copy build files over to top build directory
	@mkdir -p build/upload/
	@$(foreach var,$(BEATS),cp -r $(var)/build/upload/ build/upload/$(var)  || exit 1;)
	@cp -r build/dashboards-upload build/upload/dashboards
	@# Run tests on the generated packages.
	@go test ./dev-tools/package_test.go -files "${BUILD_DIR}/upload/*/*"

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

.PHONY: notice
notice: python-env
	@echo "Generating NOTICE"
	@$(PYTHON_ENV)/bin/python dev-tools/generate_notice.py .

# Sets up the virtual python environment
.PHONY: python-env
python-env:
	@test -d $(PYTHON_ENV) || virtualenv $(VIRTUALENV_PARAMS) $(PYTHON_ENV)
	@$(PYTHON_ENV)/bin/pip install -q --upgrade pip autopep8 six
	@# Work around pip bug. See: https://github.com/pypa/pip/issues/4464
	@find $(PYTHON_ENV) -type d -name dist-packages -exec sh -c "echo dist-packages > {}.pth" ';'

# Tests if apm works with the current code
.PHONY: test-apm
test-apm:
	sh ./script/test_apm.sh
