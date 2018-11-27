# This Makefile is for x-pack Beats. Its only responsibility is to provide
# compatibility with existing Jenkins and Travis setups.

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

.PHONY: testsuite
testsuite: mage
	mage update build unitTest integTest

.PHONY: update
update: mage
	mage update

