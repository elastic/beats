#
# gox is a tool to cross-compile Go binaries. cgo is not used when compiling.
# This is quick smoke test to ensure that nothing is broken at compile time by
# the introduction of CGO-only code.
#

#
# Variables
#
GOX_OS          ?= linux darwin windows freebsd netbsd openbsd
GOX_OSARCH      ?= !darwin/arm !darwin/arm64 !darwin/386
GOX_FLAGS       ?=
GOX_DISABLE     ?=
GOX_IMPORT_PATH ?= github.com/mitchellh/gox
GOX_PRESENT     := $(shell command -v gox 2> /dev/null)

#
# Targets
#
.PHONY: gox
gox:
ifndef GOX_DISABLE
ifndef GOX_PRESENT
	go get -u $(GOX_IMPORT_PATH)
endif
	mkdir -p build/gox
	gox -output="build/gox/{{.Dir}}-{{.OS}}-{{.Arch}}" -os="$(strip $(GOX_OS))" -osarch="$(strip $(GOX_OSARCH))" ${GOX_FLAGS}
else
	@echo gox target is disabled.
endif
