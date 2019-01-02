#
# misspell is a tool that corrects common misspellings found in files.
#

#
# Variables
#
MISSPELL_IMPORT_PATH ?= github.com/client9/misspell/cmd/misspell
MISSPELL_PRESENT     := $(shell command -v misspell 2> /dev/null)
MISSPELL_FIND        ?= find . -type f \
    -not -path "*/vendor/*" \
    -not -path "*/build/*" \
    -not -path "*/.git/*" \
    -not -path "*.json" \
    -not -path "*.log" \
    -name '*'

#
# Targets
#

.PHONY: misspell
misspell:
ifndef MISSPELL_PRESENT
	go get -u $(MISSPELL_IMPORT_PATH)
endif
	$(MISSPELL_FIND) -exec misspell -w {} \;
