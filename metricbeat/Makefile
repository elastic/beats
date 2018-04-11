# Name can be overwritten, as Metricbeat is also a library
BEAT_NAME?=metricbeat
BEAT_TITLE?=Metricbeat
BEAT_DESCRIPTION?=Metricbeat is a lightweight shipper for metrics.
SYSTEM_TESTS?=true
TEST_ENVIRONMENT?=true
ES_BEATS?=..

# Metricbeat can only be cross-compiled on platforms not requiring CGO.
GOX_OS=netbsd linux windows
GOX_FLAGS=-arch="amd64 386 arm ppc64 ppc64le"


include ${ES_BEATS}/libbeat/scripts/Makefile

# Collects all module dashboards
.PHONY: kibana
kibana:
	@rm -rf _meta/kibana
	@mkdir -p _meta/kibana
	@-cp -r module/*/_meta/kibana _meta/

# Collects all module and metricset fields
.PHONY: fields
fields: python-env
	@mkdir -p _meta
	@cp ${ES_BEATS}/metricbeat/_meta/fields.common.yml _meta/fields.generated.yml
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/metricbeat/scripts/fields_collector.py >> _meta/fields.generated.yml

# Collects all module docs
.PHONY: collect-docs
collect-docs: python-env
	@rm -rf docs/modules
	@mkdir -p docs/modules
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/metricbeat/scripts/docs_collector.py --beat ${BEAT_NAME}

# Collects all module configs
.PHONY: configs
configs: python-env
	@cp ${ES_BEATS}/metricbeat/_meta/common.yml _meta/beat.yml
	@cat ${ES_BEATS}/metricbeat/_meta/setup.yml >> _meta/beat.yml
	@cat ${ES_BEATS}/metricbeat/_meta/common.reference.yml > _meta/beat.reference.yml
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/script/config_collector.py --beat ${BEAT_NAME} --full $(PWD) >> _meta/beat.reference.yml
	@rm -rf modules.d && mkdir -p modules.d
	@for MODULE in `find module -maxdepth 1 -mindepth 1 -type d -exec basename {} \;`; do cp -a $(PWD)/module/$$MODULE/_meta/config.yml modules.d/$$MODULE.yml.disabled; done
	@chmod go-w modules.d/*
	@# Enable system by default:
	@if [ -f modules.d/system.yml.disabled ]; then mv modules.d/system.yml.disabled modules.d/system.yml; fi

# Generates imports for all modules and metricsets
.PHONY: imports
imports: python-env
	@mkdir -p include
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/script/generate_imports.py ${BEAT_PATH}

# This is called by the beats packer before building starts
.PHONY: before-build
before-build:
ifeq ($(BEAT_NAME), metricbeat)
	# disable the system/load metricset on windows
	sed -i.bk 's/- load/#- load/' $(PREFIX)/modules.d-win/system.yml
	rm $(PREFIX)/modules.d-win/system.yml.bk
	sed -i.bk 's/- load/#- load/' $(PREFIX)/metricbeat-win.reference.yml
	rm $(PREFIX)/metricbeat-win.reference.yml.bk
endif

# Runs all collection steps and updates afterwards
.PHONY: collect
collect: fields collect-docs configs kibana imports

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
	go test -tags=integration ${BEAT_PATH}/module/${MODULE}/... -v
	. ${PYTHON_ENV}/bin/activate && INTEGRATION_TESTS=1 nosetests tests/system/test_${MODULE}.py
