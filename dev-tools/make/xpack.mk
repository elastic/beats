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
include $(ES_BEATS)/dev-tools/make/mage.mk

#
# Targets (alphabetically sorted).
#
.PHONY: check
check: mage
	mage check

.PHONY: clean
clean: mage
	mage clean

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

.PHONY: testsuite
testsuite: mage
	-rm build/TEST-go-integration.out
	mage update build unitTest integTest || ( cat build/TEST-go-integration.out && false )


.PHONY: update
update: mage
	mage update

