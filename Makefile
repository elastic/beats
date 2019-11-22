BUILD_DIR=$(CURDIR)/build
COVERAGE_DIR=$(BUILD_DIR)/coverage
BEATS?=auditbeat filebeat heartbeat journalbeat metricbeat packetbeat winlogbeat x-pack/functionbeat
PROJECTS=libbeat $(BEATS)
PROJECTS_ENV=libbeat filebeat metricbeat
PYTHON_ENV?=$(BUILD_DIR)/python-env
VIRTUALENV_PARAMS?=
FIND=find . -type f -not -path "*/vendor/*" -not -path "*/build/*" -not -path "*/.git/*"
GOLINT=golint
GOLINT_REPO=golang.org/x/lint/golint
REVIEWDOG=reviewdog
REVIEWDOG_OPTIONS?=-diff "git diff master"
REVIEWDOG_REPO=github.com/haya14busa/reviewdog/cmd/reviewdog
XPACK_SUFFIX=x-pack/

# PROJECTS_XPACK_PKG is a list of Beats that have independent packaging support
# in the x-pack directory (rather than having the OSS build produce both sets
# of artifacts). This will be removed once we complete the transition.
PROJECTS_XPACK_PKG=x-pack/auditbeat x-pack/filebeat x-pack/metricbeat x-pack/winlogbeat
# PROJECTS_XPACK_MAGE is a list of Beats whose primary build logic is based in
# Mage. For compatibility with CI testing these projects support a subset of the
# makefile targets. After all Beats converge to primarily using Mage we can
# remove this and treat all sub-projects the same.
PROJECTS_XPACK_MAGE=$(PROJECTS_XPACK_PKG)

#
# Includes
#
include dev-tools/make/mage.mk

# Runs complete testsuites (unit, system, integration) for all beats with coverage and race detection.
# Also it builds the docs and the generators

.PHONY: testsuite
testsuite:
	@$(foreach var,$(PROJECTS) $(PROJECTS_XPACK_MAGE),$(MAKE) -C $(var) testsuite || exit 1;)

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
	@$(foreach var,$(PROJECTS) $(PROJECTS_XPACK_MAGE),$(MAKE) -C $(var) update || exit 1;)
	@$(MAKE) -C deploy/kubernetes all

.PHONY: clean
clean: mage
	@rm -rf build
	@$(foreach var,$(PROJECTS) $(PROJECTS_XPACK_MAGE),$(MAKE) -C $(var) clean || exit 1;)
	@$(MAKE) -C generator clean
	@-mage -clean

# Cleans up the vendor directory from unnecessary files
# This should always be run after updating the dependencies
.PHONY: clean-vendor
clean-vendor:
	@sh script/clean_vendor.sh

.PHONY: check
check: python-env
	@$(foreach var,$(PROJECTS) dev-tools $(PROJECTS_XPACK_MAGE),$(MAKE) -C $(var) check || exit 1;)
	@# Checks also python files which are not part of the beats
	@$(FIND) -name *.py -exec $(PYTHON_ENV)/bin/autopep8 -d --max-line-length 120  {} \; | (! grep . -q) || (echo "Code differs from autopep8's style" && false)
	@# Validate that all updates were committed
	@$(MAKE) update
	@$(MAKE) check-headers
	@git diff | cat
	@git update-index --refresh
	@git diff-index --exit-code HEAD --

.PHONY: check-headers
check-headers: mage
	@mage checkLicenseHeaders

.PHONY: add-headers
add-headers: mage
	@mage addLicenseHeaders

# Corrects spelling errors
.PHONY: misspell
misspell:
	go get -u github.com/client9/misspell/cmd/misspell
	# Ignore Kibana files (.json)
	$(FIND) \
		-not -path "*.json" \
		-not -path "*.log" \
		-name '*' \
		-exec misspell -w {} \;

.PHONY: fmt
fmt: add-headers python-env
	@$(foreach var,$(PROJECTS) dev-tools $(PROJECTS_XPACK_MAGE),$(MAKE) -C $(var) fmt || exit 1;)
	@# Cleans also python files which are not part of the beats
	@$(FIND) -name "*.py" -exec $(PYTHON_ENV)/bin/autopep8 --in-place --max-line-length 120 {} \;

.PHONY: lint
lint:
	@go get $(GOLINT_REPO) $(REVIEWDOG_REPO)
	$(REVIEWDOG) $(REVIEWDOG_OPTIONS)

# Builds the documents for each beat
.PHONY: docs
docs:
	@$(foreach var,$(PROJECTS),BUILD_DIR=${BUILD_DIR} $(MAKE) -C $(var) docs || exit 1;)
	sh ./script/build_docs.sh dev-guide github.com/elastic/beats/docs/devguide ${BUILD_DIR}

.PHONY: notice
notice: python-env
	@echo "Generating NOTICE"
	@$(PYTHON_ENV)/bin/python dev-tools/generate_notice.py .

# Sets up the virtual python environment
.PHONY: python-env
python-env:
	@test -d $(PYTHON_ENV) || virtualenv $(VIRTUALENV_PARAMS) $(PYTHON_ENV)
	@$(PYTHON_ENV)/bin/pip install -q --upgrade pip autopep8==1.3.5 six
	@# Work around pip bug. See: https://github.com/pypa/pip/issues/4464
	@find $(PYTHON_ENV) -type d -name dist-packages -exec sh -c "echo dist-packages > {}.pth" ';'

# Tests if apm works with the current code
.PHONY: test-apm
test-apm:
	sh ./script/test_apm.sh

### Packaging targets ####

# Builds a snapshot release.
.PHONY: snapshot
snapshot:
	@$(MAKE) SNAPSHOT=true release

# Builds a release.
.PHONY: release
release: beats-dashboards
	@$(foreach var,$(BEATS) $(PROJECTS_XPACK_PKG),$(MAKE) -C $(var) release || exit 1;)
	@$(foreach var,$(BEATS) $(PROJECTS_XPACK_PKG), \
      test -d $(var)/build/distributions && test -n "$$(ls $(var)/build/distributions)" || exit 0; \
      mkdir -p build/distributions/$(subst $(XPACK_SUFFIX),'',$(var)) && mv -f $(var)/build/distributions/* build/distributions/$(subst $(XPACK_SUFFIX),'',$(var))/ || exit 1;)

# Builds a snapshot release. The Go version defined in .go-version will be
# installed and used for the build.
.PHONY: release-manager-snapshot
release-manager-snapshot:
	@$(MAKE) SNAPSHOT=true release-manager-release

# Builds a snapshot release. The Go version defined in .go-version will be
# installed and used for the build.
.PHONY: release-manager-release
release-manager-release:
	./dev-tools/run_with_go_ver $(MAKE) release

# Collects dashboards from all Beats and generates a zip file distribution.
.PHONY: beats-dashboards
beats-dashboards: mage update
	@mage packageBeatDashboards
