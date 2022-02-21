# This is a minimal Makefile for Beats that are built with Mage. Its only
# responsibility is to provide compatibility with existing Jenkins and Travis
# setups.

#
# Variables
#
.DEFAULT_GOAL := help
PWD           := $(CURDIR)

#
# Includes
#
include $(ES_BEATS)/dev-tools/make/mage-install.mk

#
# Targets (alphabetically sorted).
#
.PHONY: check
check: mage
	mage check

.PHONY: clean
clean: mage
	mage clean

fix-permissions:

.PHONY: fmt
fmt: mage
	mage fmt

# Default target.
.PHONY: help
help:
	@echo Use mage rather than make. Here are the available mage targets:
	@mage -l

.PHONY: release
release: mage
	mage package

stop-environment:

.PHONY: unit-tests
unit-tests: mage
	mage unitTest

.PHONY: integration-tests
integration-tests: mage
	[ -e build/TEST-go-integration.out ] && rm -f build/TEST-go-integration.out || true
	mage goIntegTest || ( cat build/TEST-go-integration.out && false )

.PHONY: system-tests
system-tests: mage
	mage pythonIntegTest

.PHONY: testsuite
testsuite: mage
	[ -e build/TEST-go-integration.out ] && rm -f build/TEST-go-integration.out || true
	mage update build unitTest integTest || ( cat build/TEST-go-integration.out && false )

.PHONY: update
update: mage
	mage update

.PHONY: crosscompile
crosscompile: mage
	mage crossBuild

.PHONY: docs
docs:
	mage docs

.PHONY: docs-preview
docs-preview:
	PREVIEW=1 $(MAKE) docs
