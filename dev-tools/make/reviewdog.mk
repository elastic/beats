#
# Variables
#
REVIEWDOG_BRANCH      ?= master
REVIEWDOG_OPTIONS     ?= -diff "git diff $(REVIEWDOG_BRANCH)"
REVIEWDOG_CMD         ?= reviewdog
REVIEWDOG_IMPORT_PATH ?= github.com/haya14busa/reviewdog/cmd/reviewdog
REVIEWDOG_PRESENT     := $(shell command -v reviewdog 2> /dev/null)

GOLINT_CMD            ?= golint
GOLINT_IMPORT_PATH    ?= github.com/golang/lint/golint
GOLINT_PRESENT        := $(shell command -v golint 2> /dev/null)

#
# Targets
#

# reviewdog diffs the golint warnings between the current branch and the
# REVIEWDOG_BRANCH (defaults to master).
.PHONY: reviewdog
reviewdog:
ifndef REVIEWDOG_PRESENT
	@go get $(REVIEWDOG_IMPORT_PATH)
endif
ifndef GOLINT_PRESENT
	@go get $(GOLINT_IMPORT_PATH)
endif
	$(REVIEWDOG_CMD) $(REVIEWDOG_OPTIONS)

# lint is an alias for reviewdog.
.PHONY: lint
lint: reviewdog
