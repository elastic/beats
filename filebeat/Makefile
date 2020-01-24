BEAT_NAME?=filebeat
BEAT_TITLE?=Filebeat
SYSTEM_TESTS?=true
TEST_ENVIRONMENT?=true
GOX_FLAGS=-arch="amd64 386 arm ppc64 ppc64le"
ES_BEATS?=..
EXCLUDE_COMMON_UPDATE_TARGET=true

include ${ES_BEATS}/libbeat/scripts/Makefile

.PHONY: update
update: mage
	mage update

# Creates a new module. Requires the params MODULE.
.PHONY: create-module
create-module:
	mage createModule

# Creates a new fileset. Requires the params MODULE and FILESET.
.PHONY: create-fileset
create-fileset:
	mage createFileset

# Creates a fields.yml based on a pipeline.json file. Requires the params MODULE and FILESET.
.PHONY: create-fields
create-fields:
	mage createFields
