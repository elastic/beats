# Name can be overwritten, as Metricbeat is also a library
BEAT_NAME?=metricbeat
BEAT_TITLE?=Metricbeat
SYSTEM_TESTS?=true
TEST_ENVIRONMENT?=true
ES_BEATS?=../..

# Metricbeat can only be cross-compiled on platforms not requiring CGO.
GOX_OS=netbsd linux windows
GOX_FLAGS=-arch="amd64 386 arm ppc64 ppc64le"

DOCS_BRANCH=$(shell grep doc-branch ../../libbeat/docs/version.asciidoc | cut -c 14-)

include ${ES_BEATS}/libbeat/scripts/Makefile

# Collects all module dashboards
.PHONY: kibana
kibana:
	@rm -rf _meta/kibana.generated
	@mkdir -p _meta/kibana.generated
	@-cp -pr module/*/_meta/kibana/* _meta/kibana.generated
	@-cp -pr ../../metricbeat/module/*/_meta/kibana/* _meta/kibana.generated

# Collects all module docs
.PHONY: collect-docs
collect-docs: python-env
	@rm -rf docs/modules
	@mkdir -p docs/modules
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/metricbeat/scripts/docs_collector.py --beat ${BEAT_NAME}

# Collects all module configs
.PHONY: configs
configs: python-env
	@mkdir -p _meta
	@cp ${ES_BEATS}/metricbeat/_meta/common.yml _meta/beat.yml
	@cat ${ES_BEATS}/metricbeat/_meta/setup.yml >> _meta/beat.yml
	@cat ${ES_BEATS}/metricbeat/_meta/common.reference.yml > _meta/beat.reference.yml
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/script/config_collector.py --beat ${BEAT_NAME} --x_pack_path $(PWD) --full $(PWD)/../../metricbeat >> _meta/beat.reference.yml
	@rm -rf modules.d
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/metricbeat/scripts/modules_collector.py --docs_branch=$(DOCS_BRANCH) --is_x_pack
	@chmod go-w modules.d/*
	@# Enable system by default:
	@if [ -f modules.d/system.yml.disabled ]; then mv modules.d/system.yml.disabled modules.d/system.yml; fi

# Generates imports for all modules and metricsets
.PHONY: imports
imports: python-env
	@mkdir -p include
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/script/generate_imports.py --x_pack_path github.com/elastic/beats/x-pack/metricbeat --scripts_path ${ES_BEATS}/metricbeat/scripts ${BEAT_PATH}

# Runs all collection steps and updates afterwards
.PHONY: collect
collect: imports assets fields add-headers collect-docs configs kibana fmt

# Creates a new metricset. Requires the params MODULE and METRICSET
.PHONY: create-metricset
create-metricset: python-env
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/metricbeat/scripts/create_metricset.py --path=$(PWD) --es_beats=$(ES_BEATS) --module=$(MODULE) --metricset=$(METRICSET)

# Generates the data.json example documents
.PHONY: generate-json
generate-json: build-image
	 ${DOCKER_COMPOSE} run beat go test -tags=integration github.com/elastic/beats/metricbeat/module/... -data

.PHONY: run-module
run-module: ## @testing Runs the given module with exposing the port. Needs $MODULE and $PORT as param
run-module:
	${DOCKER_COMPOSE} build ${MODULE}
	${DOCKER_COMPOSE} run -p ${PORT}:${PORT} ${MODULE}

.PHONY: test-module
test-module: ## @testing Tests the given module. Needs $MODULE as param an run-module must be started first.
test-module: python-env update metricbeat.test
	go test -tags=integration ${ES_BEATS}/x-pack/metricbeat/module/${MODULE}/... -v
	. ${PYTHON_ENV}/bin/activate && INTEGRATION_TESTS=1 nosetests tests/system/test_${MODULE}.py

.PHONY: assets
assets:
	go run ${ES_BEATS}/metricbeat/scripts/assets/assets.go ${ES_BEATS}/x-pack/metricbeat/module Elastic
	mkdir -p include/fields
	go run  ${ES_BEATS}/dev-tools/cmd/asset/asset.go -license Elastic -pkg include -in ${ES_BEATS}/metricbeat/_meta/fields.common.yml -out include/fields.go $(BEAT_NAME)
	# go run  ${ES_BEATS}/libbeat/scripts/cmd/global_fields/main.go -es_beats_path ${ES_BEATS} -beat_path ${PWD} | go run  ${ES_BEATS}/dev-tools/cmd/asset/asset.go -license Elastic -out ./include/fields/fields.go -pkg include ${ES_BEATS}/libbeat/fields.yml $(BEAT_NAME)

# Overwrite check-headers - check for Apache license in
# auditbeat/ and Elastic license in xpack/auditbeat/
.PHONY: check-headers
check-headers:
ifndef CHECK_HEADERS_DISABLED
	@go get -u github.com/elastic/go-licenser
	@go-licenser -d -license ${LICENSE} ${ES_BEATS}/${BEAT_NAME}
	@go-licenser -d -license Elastic .
endif

# Overwrite check-headers - insert Apache license in
# auditbeat/ and Elastic license in xpack/{beat}/
.PHONY: add-headers
add-headers:
ifndef CHECK_HEADERS_DISABLED
	@go get github.com/elastic/go-licenser
	@go-licenser -license ${LICENSE} ${ES_BEATS}/x-pack/${BEAT_NAME}
	@go-licenser -license Elastic .
endif

.PHONY: fields
fields:
	@mage fields

#.PHONY: update
#update: python-env fields collect
#	@echo "Updating generated files for ${BEAT_NAME}"
#
#	@mkdir -p _meta
#	@# Update config files.
#	@cat _meta/beat.yml ${ES_BEATS}/libbeat/_meta/config.yml | sed -e "s/beatname/${BEAT_NAME}/g;s/beat-index-prefix/${BEAT_INDEX_PREFIX}/g" > ${BEAT_NAME}.yml
#	@chmod 0640 ${BEAT_NAME}.yml
