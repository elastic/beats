MAGE_VERSION     ?= $(shell grep 'magefile' go.mod | cut -d' ' -f2)
MAGE_PRESENT     := $(shell mage --version 2> /dev/null | grep $(MAGE_VERSION))
MAGE_IMPORT_PATH ?= github.com/magefile/mage
export MAGE_IMPORT_PATH

.PHONY: mage
mage:
ifndef MAGE_PRESENT
	@echo Installing mage $(MAGE_VERSION).
	@go install -ldflags="-X $(MAGE_IMPORT_PATH)/mage.gitTag=$(MAGE_VERSION)" ${MAGE_IMPORT_PATH}
	@-mage -clean
endif
	@true
