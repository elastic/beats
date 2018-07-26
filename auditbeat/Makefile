BEAT_NAME=auditbeat
BEAT_TITLE=Auditbeat
SYSTEM_TESTS=true
TEST_ENVIRONMENT?=true
GOX_OS?=linux windows ## @Building List of all OS to be supported by "make crosscompile".
DEV_OS?=linux
TESTING_ENVIRONMENT?=snapshot-noxpack
ES_BEATS?=..

# Path to the libbeat Makefile
include ${ES_BEATS}/libbeat/scripts/Makefile

# Collects all dependencies and then calls update
.PHONY: collect
collect: collect-docs configs kibana

# Collects all module configs
.PHONY: configs
configs: python-env
	@cat ${ES_BEATS}/auditbeat/_meta/common.p1.yml \
		<(go run scripts/generate_config.go -os ${DEV_OS} -concat) \
		${ES_BEATS}/auditbeat/_meta/common.p2.yml > _meta/beat.yml
	@cat ${ES_BEATS}/auditbeat/_meta/common.reference.yml \
		<(go run scripts/generate_config.go -os ${DEV_OS} -ref -concat) > _meta/beat.reference.yml

# Collects all module docs
.PHONY: collect-docs
collect-docs: python-env
	@rm -rf docs/modules
	@mkdir -p docs/modules
	@go run scripts/generate_config.go -os linux
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/auditbeat/scripts/docs_collector.py --beat ${BEAT_NAME}

# Collects all module dashboards
.PHONY: kibana
kibana:
	@-rm -rf _meta/kibana.generated
	@mkdir -p _meta/kibana.generated
	@-cp -pr module/*/_meta/kibana/* _meta/kibana.generated
