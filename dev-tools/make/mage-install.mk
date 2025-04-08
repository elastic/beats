MAGE_IMPORT_PATH ?= github.com/magefile/mage
MAGE_VERSION     ?= $(shell go list -m -f '{{.Version}}' $(MAGE_IMPORT_PATH))
MAGE_PRESENT     := $(shell mage --version 2> /dev/null | grep $(MAGE_VERSION))
export MAGE_IMPORT_PATH

.PHONY: mage
mage:
ifndef MAGE_PRESENT
	@echo Installing mage $(MAGE_VERSION).
	@go install -ldflags="-X $(MAGE_IMPORT_PATH)/mage.gitTag=$(MAGE_VERSION)" ${MAGE_IMPORT_PATH}
	@-mage -clean
endif
	@true
