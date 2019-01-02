#
# Variables
#
.DEFAULT_GOAL := help

#
# Includes
#
include dev-tools/make/mage.mk
include dev-tools/make/misspell.mk
include dev-tools/make/reviewdog.mk

#
# Targets (sorted alphabetically)
#

# Collects dashboards from all Beats and generates a zip file distribution.
.PHONY: beats-dashboards
beats-dashboards: mage
	mage package:dashboards

.PHONY: check
check: mage
	mage check

.PHONY: clean
clean: mage
	mage clean

# Cleans up the vendor directory from unnecessary files
# This should always be run after updating the dependencies
.PHONY: clean-vendor
clean-vendor:
	@sh script/clean_vendor.sh

.PHONY: docs
docs: mage
	mage docs

.PHONY: fmt
fmt: mage
	mage fmt

# Default target.
.PHONY: help
help:
	@echo Use mage rather than make. Here are the available mage targets:
	@mage -l

# Check Makefile format.
.PHONY: makelint
makelint: SHELL:=/bin/bash
makelint:
	@diff <(grep ^.PHONY Makefile | sort) <(grep ^.PHONY Makefile) \
	  || echo Makefile targets need to be sorted.

.PHONY: notice
notice: mage
	mage update:notice

# Builds a release.
.PHONY: release
release: mage
	mage package

# Builds a snapshot release. The Go version defined in .go-version will be
# installed and used for the build.
.PHONY: release-manager-release
release-manager-release:
	./dev-tools/run_with_go_ver $(MAKE) release

# Builds a snapshot release. The Go version defined in .go-version will be
# installed and used for the build.
.PHONY: release-manager-snapshot
release-manager-snapshot:
	@$(MAKE) SNAPSHOT=true release-manager-release

.PHONY: setup-commit-hook
setup-commit-hook:
	@cp script/pre_commit.sh .git/hooks/pre-commit
	@chmod 751 .git/hooks/pre-commit

# Builds a snapshot release.
.PHONY: snapshot
snapshot:
	@$(MAKE) SNAPSHOT=true release

# Tests if apm works with the current code
.PHONY: test-apm
test-apm:
	sh ./script/test_apm.sh

.PHONY: testsuite
testsuite: mage
	mage test:all

.PHONY: update
update: mage
	mage update
