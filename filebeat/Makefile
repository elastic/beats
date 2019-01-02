#
# Variables
#
GOX_FLAGS := -arch="amd64 386 arm ppc64 ppc64le"

#
# Includes
#
include ../dev-tools/make/oss.mk

#
# Targets
#

# Creates a new module. Requires the params MODULE.
.PHONY: create-module
create-module:
	@go run ${ES_BEATS}/filebeat/scripts/generator/module/main.go --path=$(PWD) --beats_path=$(BEAT_GOPATH)/src/$(BEAT_PATH) --module=$(MODULE)

# Creates a new fileset. Requires the params MODULE and FILESET.
.PHONY: create-fileset
create-fileset:
	@go run ${ES_BEATS}/filebeat/scripts/generator/fileset/main.go --path=$(PWD) --beats_path=$(BEAT_GOPATH)/src/$(BEAT_PATH) --module=$(MODULE) --fileset=$(FILESET)

# Creates a fields.yml based on a pipeline.json file. Requires the params MODULE and FILESET.
.PHONY: create-fields
create-fields:
	@go run ${ES_BEATS}/filebeat/scripts/generator/fields/main.go --beats_path=$(BEAT_GOPATH)/src/$(BEAT_PATH) --module=$(MODULE) --fileset=$(FILESET)
