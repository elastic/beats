BUILD_DIR=$(CURDIR)/build
COVERAGE_DIR=$(BUILD_DIR)/coverage
BEATS?=auditbeat filebeat heartbeat metricbeat packetbeat winlogbeat x-pack/functionbeat x-pack/osquerybeat
PROJECTS=libbeat $(BEATS)
PROJECTS_ENV=libbeat filebeat metricbeat
PYTHON_ENV?=$(BUILD_DIR)/python-env
PYTHON_EXE?=python3
PYTHON_ENV_EXE=${PYTHON_ENV}/bin/$(notdir ${PYTHON_EXE})
VENV_PARAMS?=
FIND=find . -type f -not -path "*/build/*" -not -path "*/.git/*"
XPACK_SUFFIX=x-pack/

BEAT_VERSION=$(shell grep defaultBeatVersion libbeat/version/version.go | cut -d'=' -f2 | tr -d '" ')

# PROJECTS_XPACK_PKG is a list of Beats that have independent packaging support
# in the x-pack directory (rather than having the OSS build produce both sets
# of artifacts). This will be removed once we complete the transition.
PROJECTS_XPACK_PKG=x-pack/auditbeat x-pack/dockerlogbeat x-pack/filebeat x-pack/heartbeat x-pack/metricbeat x-pack/winlogbeat x-pack/packetbeat
# PROJECTS_XPACK_MAGE is a list of Beats whose primary build logic is based in
# Mage. For compatibility with CI testing these projects support a subset of the
# makefile targets. After all Beats converge to primarily using Mage we can
# remove this and treat all sub-projects the same.
PROJECTS_XPACK_MAGE=$(PROJECTS_XPACK_PKG) x-pack/libbeat

#
# Includes
#
include dev-tools/make/mage-install.mk

## help : Show this help.
help: Makefile
	@printf "Usage: make [target] [VARIABLE=value]\nTargets:\n"
	@sed -n 's/^## //p' $< | awk 'BEGIN {FS = ":"}; { if(NF>1 && $$2!="") printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 ; else printf "%40s\n", $$1};'
	@printf "Variables:\n"
	@grep -E "^[A-Za-z0-9_]*\?=" $< | awk 'BEGIN {FS = "\\?="}; { printf "  \033[36m%-25s\033[0m  Default values: %s\n", $$1, $$2}'


## testsuite : Runs complete testsuites (unit, system, integration) for all beats with coverage and race detection. Also it builds the docs and the generators.
.PHONY: testsuite
testsuite:
	@$(foreach var,$(PROJECTS) $(PROJECTS_XPACK_MAGE),$(MAKE) -C $(var) testsuite || exit 1;)

## setup-commit-hook : Setup the git pre-commit hook
.PHONY: setup-commit-hook
setup-commit-hook:
	@cp script/pre_commit.sh .git/hooks/pre-commit
	@chmod 751 .git/hooks/pre-commit

## stop-environments : Stop the environment for each project.
stop-environments:
	@$(foreach var,$(PROJECTS_ENV),$(MAKE) -C $(var) stop-environment || exit 0;)

## test : Runs unit and system tests without coverage and race detection.
.PHONY: test
test:
	@$(foreach var,$(PROJECTS),$(MAKE) -C $(var) test || exit 1;)

## unit : Runs unit tests without coverage and race detection.
.PHONY: unit
unit:
	@$(foreach var,$(PROJECTS),$(MAKE) -C $(var) unit || exit 1;)

## crosscompile : Crosscompile all beats.
.PHONY: crosscompile
crosscompile:
	@$(foreach var,filebeat winlogbeat metricbeat heartbeat auditbeat,$(MAKE) -C $(var) crosscompile || exit 1;)

## coverage-report : Generates coverage report.
.PHONY: coverage-report
coverage-report:
	@mkdir -p $(COVERAGE_DIR)
	@echo 'mode: atomic' > ./$(COVERAGE_DIR)/full.cov
	@# Collects all coverage files and skips top line with mode
	@$(foreach var,$(PROJECTS),tail -q -n +2 ./$(var)/$(COVERAGE_DIR)/*.cov >> ./$(COVERAGE_DIR)/full.cov || true;)
	@go tool cover -html=./$(COVERAGE_DIR)/full.cov -o $(COVERAGE_DIR)/full.html
	@echo "Generated coverage report $(COVERAGE_DIR)/full.html"

## update : TBD.
.PHONY: update
update: notice
	@$(foreach var,$(PROJECTS) $(PROJECTS_XPACK_MAGE),$(MAKE) -C $(var) update || exit 1;)
	@$(MAKE) -C deploy/kubernetes all

## clean : Clean target.
.PHONY: clean
clean: mage
	@rm -rf build
	@$(foreach var,$(PROJECTS) $(PROJECTS_XPACK_MAGE),$(MAKE) -C $(var) clean || exit 1;)
	@$(MAKE) -C generator clean
	@-mage -clean

## check : TBD.
.PHONY: check
check:
	@$(foreach var,$(PROJECTS) dev-tools $(PROJECTS_XPACK_MAGE),$(MAKE) -C $(var) check || exit 1;)
	$(MAKE) check-python
	# check if vendor folder does not exists
	[ ! -d vendor ]
	# Validate that all updates were committed
	@$(MAKE) update
	@$(MAKE) check-headers
	@$(MAKE) check-go
	@$(MAKE) check-no-changes

## check : Run some checks similar to what the default check validation runs in the CI.
.PHONY: check-default
check-default:
	@$(MAKE) check-python
	@echo "The update goal is skipped to speed up the checks in the CI on a PR basis."
	@$(MAKE) notice
	@$(MAKE) check-headers
	@$(MAKE) check-go
	@$(MAKE) check-no-changes

## ccheck-go : Check there is no changes in Go modules.
.PHONY: check-go
check-go:
	@go mod tidy

## ccheck-no-changes : Check there is no local changes.
.PHONY: check-no-changes
check-no-changes:
	@go mod tidy
	@git diff | cat
	@git update-index --refresh
	@git diff-index --exit-code HEAD --

## check-python : Python Linting.
.PHONY: check-python
check-python: python-env
	@. $(PYTHON_ENV)/bin/activate; \
	$(FIND) -name *.py -name *.py -not -path "*/build/*" -exec $(PYTHON_ENV)/bin/autopep8 -d --max-line-length 120  {} \; | (! grep . -q) || (echo "Code differs from autopep8's style" && false); \
	$(FIND) -name *.py -not -path "*/build/*" | xargs $(PYTHON_ENV)/bin/pylint --py3k -E || (echo "Code is not compatible with Python 3" && false)

## check-headers : Check the license headers.
.PHONY: check-headers
check-headers: mage
	@mage checkLicenseHeaders

## add-headers : Adds the license headers.
.PHONY: add-headers
add-headers: mage
	@mage addLicenseHeaders

## misspell : Corrects spelling errors.
.PHONY: misspell
misspell:
	go get -u github.com/client9/misspell/cmd/misspell
	# Ignore Kibana files (.json)
	$(FIND) \
		-not -path "*.json" \
		-not -path "*.log" \
		-name '*' \
		-exec misspell -w {} \;

## fmt : TBD.
.PHONY: fmt
fmt: add-headers python-env
	@$(foreach var,$(PROJECTS) dev-tools $(PROJECTS_XPACK_MAGE),$(MAKE) -C $(var) fmt || exit 1;)
	@# Cleans also python files which are not part of the beats
	@$(FIND) -name "*.py" -exec $(PYTHON_ENV)/bin/autopep8 --in-place --max-line-length 120 {} \;

## docs : Builds the documents for each beat
.PHONY: docs
docs:
	@$(foreach var,$(PROJECTS),BUILD_DIR=${BUILD_DIR} $(MAKE) -C $(var) docs || exit 1;)
	sh ./script/build_docs.sh dev-guide github.com/elastic/beats/docs/devguide ${BUILD_DIR}

## notice : Generates the NOTICE file.
.PHONY: notice
notice:
	@echo "Generating NOTICE"
	go mod tidy
	go mod download
	go list -m -json all | go run go.elastic.co/go-licence-detector \
		-includeIndirect \
		-rules dev-tools/notice/rules.json \
		-overrides dev-tools/notice/overrides.json \
		-noticeTemplate dev-tools/notice/NOTICE.txt.tmpl \
		-noticeOut NOTICE.txt \
		-depsOut ""


## python-env : Sets up the virtual python environment.
.PHONY: python-env
python-env:
	@test -f $(PYTHON_ENV)/bin/activate || ${PYTHON_EXE} -m venv $(VENV_PARAMS) $(PYTHON_ENV)
	@. $(PYTHON_ENV)/bin/activate; \
	${PYTHON_EXE} -m pip install -q --upgrade pip autopep8==1.5.4 pylint==2.4.4; \
	find $(PYTHON_ENV) -type d -name dist-packages -exec sh -c "echo dist-packages > {}.pth" ';'
	@# Work around pip bug. See: https://github.com/pypa/pip/issues/4464

## test-apm : Tests if apm works with the current code
.PHONY: test-apm
test-apm:
	sh ./script/test_apm.sh

## get-version : Get the libbeat version
.PHONY: get-version
get-version:
	@echo $(BEAT_VERSION)

### Packaging targets ####

## snapshot : Builds a snapshot release.
.PHONY: snapshot
snapshot:
	@$(MAKE) SNAPSHOT=true release

## release : Builds a release.
.PHONY: release
release: beats-dashboards
	@mage dumpVariables
	@$(foreach var,$(BEATS) $(PROJECTS_XPACK_PKG),$(MAKE) -C $(var) release || exit 1;)
	@$(foreach var,$(BEATS) $(PROJECTS_XPACK_PKG), \
      test -d $(var)/build/distributions && test -n "$$(ls $(var)/build/distributions)" || exit 0; \
      mkdir -p build/distributions/$(subst $(XPACK_SUFFIX),'',$(var)) && mv -f $(var)/build/distributions/* build/distributions/$(subst $(XPACK_SUFFIX),'',$(var))/ || exit 1;)

## release-manager-snapshot : Builds a snapshot release. The Go version defined in .go-version will be installed and used for the build.
.PHONY: release-manager-snapshot
release-manager-snapshot:
	@$(MAKE) SNAPSHOT=true release-manager-release

## release-manager-release : Builds a snapshot release. The Go version defined in .go-version will be installed and used for the build.
.PHONY: release-manager-release
release-manager-release:
	GO_VERSION=$(shell cat ./.go-version) ./dev-tools/run_with_go_ver $(MAKE) release

## beats-dashboards : Collects dashboards from all Beats and generates a zip file distribution.
.PHONY: beats-dashboards
beats-dashboards: mage update
	@mage packageBeatDashboards

## build/distributions/dependencies.csv : Generates the dependencies file
build/distributions/dependencies.csv: $(PYTHON)
	@mkdir -p build/distributions
	$(PYTHON) dev-tools/dependencies-report --csv $@
