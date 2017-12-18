BEAT_NAME?=filebeat
BEAT_TITLE?=Filebeat
BEAT_DESCRIPTION?=Filebeat sends log files to Logstash or directly to Elasticsearch.
SYSTEM_TESTS?=true
TEST_ENVIRONMENT?=true
GOX_FLAGS=-arch="amd64 386 arm ppc64 ppc64le"
ES_BEATS?=..

include ${ES_BEATS}/libbeat/scripts/Makefile

# This is called by the beats packer before building starts
.PHONY: before-build
before-build:

# Collects all module dashboards
.PHONY: kibana
kibana:
	@rm -rf _meta/kibana
	@mkdir -p _meta/kibana
	@-cp -r module/*/_meta/kibana _meta/

# Collects all module and dataset fields
.PHONY: fields
fields:
	@mkdir -p _meta/
	@cp ${ES_BEATS}/filebeat/_meta/fields.common.yml  _meta/fields.generated.yml
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/metricbeat/scripts/fields_collector.py >> _meta/fields.generated.yml

# Collects all modules files to be packaged in a temporary folder
.PHONY: modules
modules:
	@mkdir -p _meta/
	@rm -rf _meta/module.generated
	@rsync -q -av module/ _meta/module.generated --exclude "_meta" --exclude "*/*/test"

# Collects all module configs
.PHONY: configs
configs: python-env
	@cp ${ES_BEATS}/filebeat/_meta/common.p1.yml _meta/beat.yml
	@cat ${ES_BEATS}/filebeat/_meta/common.p2.yml >> _meta/beat.yml
	@cat ${ES_BEATS}/filebeat/_meta/common.reference.p1.yml > _meta/beat.reference.yml
	@${PYTHON_ENV}/bin/python  ${ES_BEATS}/script/config_collector.py --beat ${BEAT_NAME} --full $(PWD) >> _meta/beat.reference.yml
	@cat ${ES_BEATS}/filebeat/_meta/common.reference.p2.yml >> _meta/beat.reference.yml
	@rm -rf modules.d && mkdir -p modules.d
	@for MODULE in `find module -type d -maxdepth 1 -mindepth 1 -exec basename {} \;`; do cp -a $(PWD)/module/$$MODULE/_meta/config.yml modules.d/$$MODULE.yml.disabled; done
	@chmod go-w modules.d/*

# Collects all module docs
.PHONY: collect-docs
collect-docs: python-env
	@rm -rf docs/modules
	@mkdir -p docs/modules
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/filebeat/scripts/docs_collector.py --beat ${BEAT_NAME}

# Generate imports for prospectors
.PHONY: imports
imports: python-env
	@mkdir -p include
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/script/generate_imports.py ${BEAT_PATH}

# Runs all collection steps and updates afterwards
.PHONY: collect
collect: fields kibana modules configs collect-docs imports

# Creates a new fileset. Requires the params MODULE and FILESET
.PHONY: create-fileset
create-fileset: python-env
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/filebeat/scripts/create_fileset.py --path=$(PWD) --es_beats=$(ES_BEATS) --module=$(MODULE) --fileset=$(FILESET)

