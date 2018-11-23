MAGE_PRESENT := $(shell command -v mage 2> /dev/null)
MAGE_IMPORT_PATH?=github.com/elastic/beats/vendor/github.com/magefile/mage
export MAGE_IMPORT_PATH

.PHONY: mage
mage:
ifndef MAGE_PRESENT
	go install ${MAGE_IMPORT_PATH}
	@-mage -clean 2> /dev/null
endif
	@true
