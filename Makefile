BUILD_DIR=$(CURDIR)/build
COVERAGE_DIR=$(BUILD_DIR)/coverage
BEATS?=auditbeat filebeat heartbeat metricbeat packetbeat winlogbeat x-pack/auditbeat x-pack/dockerlogbeat x-pack/filebeat x-pack/heartbeat x-pack/metricbeat x-pack/osquerybeat x-pack/packetbeat x-pack/winlogbeat
PROJECTS=libbeat x-pack/libbeat $(BEATS)
PROJECTS_ENV=libbeat filebeat metricbeat
PYTHON_ENV?=$(BUILD_DIR)/python-env

# Python version detection with caching for speed
# Checks if default python3 is compatible first, only searches alternatives if needed
PYTHON_CACHE_FILE=$(BUILD_DIR)/.python_exe_cache
PYTHON_EXE?=$(shell \
	if [ -f "$(PYTHON_CACHE_FILE)" ]; then cat "$(PYTHON_CACHE_FILE)"; \
	else \
		py=python3; \
		ver=$$($$py -c 'import sys; print(sys.version_info.minor)' 2>/dev/null || echo 99); \
		if [ "$$ver" -ge 9 ] && [ "$$ver" -lt 14 ]; then echo $$py; \
		else \
			for py in python3.13 python3.12 python3.11 python3.10 \
				/opt/homebrew/opt/python@3.13/bin/python3.13 \
				/opt/homebrew/opt/python@3.12/bin/python3.12; do \
				if $$py -c 'import sys; exit(0 if 9<=sys.version_info.minor<14 else 1)' 2>/dev/null; then \
					echo $$py; break; \
				fi; \
			done; \
		fi; \
	fi)
ifeq ($(PYTHON_EXE),)
PYTHON_EXE=python3
endif

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

#
# Pre-compiled dev tools for faster builds
#
DEV_TOOLS_BIN=$(BUILD_DIR)/dev-tools-bin

## dev-tools-bin : Pre-compiles dev tools for faster builds.
.PHONY: dev-tools-bin
dev-tools-bin:
	@mkdir -p $(DEV_TOOLS_BIN)
	@for tool in asset module_fields module_include_list; do \
		src="dev-tools/cmd/$${tool}/$${tool}.go"; \
		bin="$(DEV_TOOLS_BIN)/$${tool}"; \
		if [ ! -f "$$bin" ] || [ "$$src" -nt "$$bin" ]; then \
			echo ">> Compiling $$src..."; \
			go build -o "$$bin" "./$${src}"; \
		fi; \
	done

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

## update : Updates all beats. Use UPDATE_PARALLEL=1 for parallel builds (faster).
# OSS beats that don't have x-pack counterparts running in parallel
OSS_BEATS_ONLY=heartbeat packetbeat winlogbeat
# OSS beats with x-pack counterparts (must not run in parallel with x-pack version)
OSS_BEATS_WITH_XPACK=auditbeat filebeat metricbeat
# X-pack only beats
XPACK_BEATS_ONLY=x-pack/agentbeat x-pack/dockerlogbeat x-pack/osquerybeat

.PHONY: update
update: notice
ifeq ($(UPDATE_PARALLEL),1)
	@JOBS=$$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4); \
	START=$$(date +%s); \
	echo ">> Running parallel update ($$JOBS jobs)..."; \
	go build ./dev-tools/mage/... 2>/dev/null || true; \
	echo ">> Step 1/3: Updating libbeat (in parallel)..."; \
	echo "libbeat x-pack/libbeat" | tr ' ' '\n' | \
		xargs -P 2 -I {} sh -c '$(MAKE) -C {} update || exit 255' && \
	echo ">> Step 2/3: Updating OSS beats + x-pack-only beats in parallel..." && \
	echo "$(OSS_BEATS_ONLY) $(OSS_BEATS_WITH_XPACK) $(XPACK_BEATS_ONLY)" | tr ' ' '\n' | \
		xargs -P $$JOBS -I {} sh -c '$(MAKE) -C {} update || exit 255' && \
	echo ">> Step 3/3: Updating x-pack beats with OSS counterparts + Kubernetes manifests (in parallel)..." && \
	( echo "x-pack/auditbeat x-pack/filebeat x-pack/heartbeat x-pack/metricbeat x-pack/packetbeat x-pack/winlogbeat" | tr ' ' '\n' | \
		xargs -P $$JOBS -I {} sh -c '$(MAKE) -C {} update || exit 255' ) & \
	XPACK_PID=$$!; \
	$(MAKE) -C deploy/kubernetes all >/dev/null 2>&1 & \
	K8S_PID=$$!; \
	wait $$XPACK_PID $$K8S_PID && \
	echo ">> Complete!"
else
	@$(foreach var,$(PROJECTS) $(PROJECTS_XPACK_MAGE),$(MAKE) -C $(var) update || exit 1;)
	@$(MAKE) -C deploy/kubernetes all
endif

## update-fast : Parallel update with caching optimizations.
.PHONY: update-fast
update-fast: dev-tools-bin
	@$(MAKE) UPDATE_PARALLEL=1 SKIP_NOTICE=1 update

## clean : Clean target. Use CLEAN_PARALLEL=1 for parallel clean (faster).
.PHONY: clean
clean: mage
	@rm -rf build
ifeq ($(CLEAN_PARALLEL),1)
	@echo "$(PROJECTS) $(PROJECTS_XPACK_MAGE)" | tr ' ' '\n' | \
		xargs -P $$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4) -I {} \
		sh -c '$(MAKE) -C {} clean 2>/dev/null || true'
else
	@$(foreach var,$(PROJECTS) $(PROJECTS_XPACK_MAGE),$(MAKE) -C $(var) clean || exit 1;)
endif
	@-mage -clean

## clean-fast : Alias for parallel clean.
.PHONY: clean-fast
clean-fast:
	@$(MAKE) CLEAN_PARALLEL=1 clean

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

## go-mod-tidy : Runs go mod tidy with caching to avoid redundant runs.
# Uses a marker file to skip if go.mod/go.sum haven't changed since last tidy.
GO_MOD_TIDY_MARKER=$(BUILD_DIR)/.go_mod_tidy_done
.PHONY: go-mod-tidy
go-mod-tidy:
	@mkdir -p $(BUILD_DIR)
	@if [ ! -f "$(GO_MOD_TIDY_MARKER)" ] || \
	   [ go.mod -nt "$(GO_MOD_TIDY_MARKER)" ] || \
	   [ go.sum -nt "$(GO_MOD_TIDY_MARKER)" ]; then \
		echo ">> Running go mod tidy..."; \
		go mod tidy && touch "$(GO_MOD_TIDY_MARKER)"; \
	else \
		echo ">> go mod tidy is up-to-date (skipping)"; \
	fi

## check-go : Check there is no changes in Go modules.
.PHONY: check-go
check-go: go-mod-tidy

## check-no-changes : Check there is no local changes.
.PHONY: check-no-changes
check-no-changes: go-mod-tidy
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

## fmt : Formats code. Use FMT_PARALLEL=1 for parallel formatting (faster).
.PHONY: fmt
fmt: add-headers python-env
ifeq ($(FMT_PARALLEL),1)
	@JOBS=$$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4); \
	echo ">> Running parallel fmt ($$JOBS jobs)..."; \
	echo "$(PROJECTS) dev-tools $(PROJECTS_XPACK_MAGE)" | tr ' ' '\n' | \
		xargs -P $$JOBS -I {} sh -c '$(MAKE) -C {} fmt 2>/dev/null || true'
else
	@$(foreach var,$(PROJECTS) dev-tools $(PROJECTS_XPACK_MAGE),$(MAKE) -C $(var) fmt || exit 1;)
endif
	@# Cleans also python files which are not part of the beats
	@$(FIND) -name "*.py" -exec $(PYTHON_ENV)/bin/autopep8 --in-place --max-line-length 120 {} \;

## fmt-fast : Alias for parallel fmt.
.PHONY: fmt-fast
fmt-fast:
	@$(MAKE) FMT_PARALLEL=1 fmt

## docs : Builds the documents for each beat
.PHONY: docs
docs:
	@$(foreach var,$(PROJECTS),BUILD_DIR=${BUILD_DIR} $(MAKE) -C $(var) docs || exit 1;)
	sh ./script/build_docs.sh dev-guide github.com/elastic/beats/docs/devguide ${BUILD_DIR}

## notice : Generates the NOTICE file. Use SKIP_NOTICE=1 to skip if up-to-date.
.PHONY: notice
notice: go-mod-tidy
ifeq ($(SKIP_NOTICE),1)
	@if [ -f NOTICE.txt ] && [ NOTICE.txt -nt go.mod ] && [ NOTICE.txt -nt go.sum ]; then \
		echo ">> NOTICE.txt is up-to-date (skipping)"; \
	else \
		echo "Generating NOTICE"; \
		go mod download && \
		go list -m -json all | go run go.elastic.co/go-licence-detector \
			-includeIndirect \
			-rules dev-tools/notice/rules.json \
			-overrides dev-tools/notice/overrides.json \
			-noticeTemplate dev-tools/notice/NOTICE.txt.tmpl \
			-noticeOut NOTICE.txt \
			-depsOut ""; \
	fi
else
	@echo "Generating NOTICE"
	go mod download
	go list -m -json all | go run go.elastic.co/go-licence-detector \
		-includeIndirect \
		-rules dev-tools/notice/rules.json \
		-overrides dev-tools/notice/overrides.json \
		-noticeTemplate dev-tools/notice/NOTICE.txt.tmpl \
		-noticeOut NOTICE.txt \
		-depsOut ""
endif


## python-env : Sets up the virtual python environment.
.PHONY: python-env
python-env:
	@mkdir -p $(BUILD_DIR)
	@echo "$(PYTHON_EXE)" > $(PYTHON_CACHE_FILE)
	@test -f $(PYTHON_ENV)/bin/activate || ${PYTHON_EXE} -m venv $(VENV_PARAMS) $(PYTHON_ENV)
	@. $(PYTHON_ENV)/bin/activate; \
	${PYTHON_EXE} -m pip install -q --upgrade pip autopep8==1.5.4 pylint==2.4.4; \
	find $(PYTHON_ENV) -type d -name dist-packages -exec sh -c "echo dist-packages > {}.pth" ';'
	@# Work around pip bug. See: https://github.com/pypa/pip/issues/4464

## clean-python-cache : Clears the cached Python executable path (forces re-detection)
.PHONY: clean-python-cache
clean-python-cache:
	@rm -f $(PYTHON_CACHE_FILE)

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

## beats-dashboards : Collects dashboards from all Beats and generates a zip file distribution.
.PHONY: beats-dashboards
beats-dashboards: mage update
	@mage packageBeatDashboards

## build/distributions/dependencies.csv : Generates the dependencies file
build/distributions/dependencies.csv: $(PYTHON)
	@mkdir -p build/distributions
	$(PYTHON) dev-tools/dependencies-report --csv $@

## test-mage : Test the mage installation used by the Unified Release process
test-mage: mage
	@mage dumpVariables
