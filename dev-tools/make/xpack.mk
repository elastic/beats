# This Makefile is for x-pack Beats. Its only responsibility is to provide
# compatibility with existing Jenkins and Travis setups.

#
# Variables
#
PWD           := $(CURDIR)
.DEFAULT_GOAL := help

#
# Includes
#
include $(ES_BEATS)/dev-tools/make/mage.mk

#
# Targets
#
.PHONY: clean
clean: mage
	mage clean

.PHONY: fmt
fmt: mage
	mage fmt

.PHONY: check
check: mage
	mage check

.PHONY: testsuite
testsuite: mage
	mage update build unitTest integTest

# Default target.
.PHONY: help
help:
	@echo Use mage rather than make. Here are the available mage targets:
	@mage -l
