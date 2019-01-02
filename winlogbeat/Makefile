#
# Variables
#
GOX_OS   := windows

#
# Includes
#
include ../dev-tools/make/oss.mk

#
# Targets
#

.PHONY: gen
gen:
	GOOS=windows GOARCH=386 go generate -v -x ./...
