BEAT_NAME=auditbeat
BEAT_TITLE=Auditbeat
BEAT_DESCRIPTION=Audit the activities of users and processes on your system.
SYSTEM_TESTS=false
TEST_ENVIRONMENT=false

# Path to the libbeat Makefile
-include ../libbeat/scripts/Makefile

# This is called by the beats packer before building starts
.PHONY: before-build
before-build:
	@cat ${ES_BEATS}/auditbeat/_meta/common.p1.yml \
	     <(go run scripts/generate_config.go -os windows -concat) \
	     ${ES_BEATS}/auditbeat/_meta/common.p2.yml \
	     ${ES_BEATS}/libbeat/_meta/config.yml > \
	     ${PREFIX}/${BEAT_NAME}-win.yml
	@cat ${ES_BEATS}/auditbeat/_meta/common.reference.yml \
	     <(go run scripts/generate_config.go -os windows -concat) \
	     ${ES_BEATS}/libbeat/_meta/config.reference.yml > \
	     ${PREFIX}/${BEAT_NAME}-win.reference.yml

	@cat ${ES_BEATS}/auditbeat/_meta/common.p1.yml \
	     <(go run scripts/generate_config.go -os darwin -concat) \
	     ${ES_BEATS}/auditbeat/_meta/common.p2.yml \
	     ${ES_BEATS}/libbeat/_meta/config.yml > \
	     ${PREFIX}/${BEAT_NAME}-darwin.yml
	@cat ${ES_BEATS}/auditbeat/_meta/common.reference.yml \
	     <(go run scripts/generate_config.go -os darwin -concat) \
	     ${ES_BEATS}/libbeat/_meta/config.reference.yml > \
	     ${PREFIX}/${BEAT_NAME}-darwin.reference.yml

	@cat ${ES_BEATS}/auditbeat/_meta/common.p1.yml \
	     <(go run scripts/generate_config.go -os linux -concat) \
	     ${ES_BEATS}/auditbeat/_meta/common.p2.yml \
	     ${ES_BEATS}/libbeat/_meta/config.yml > \
	     ${PREFIX}/${BEAT_NAME}-linux.yml
	@cat ${ES_BEATS}/auditbeat/_meta/common.reference.yml \
	     <(go run scripts/generate_config.go -os linux -concat) \
	     ${ES_BEATS}/libbeat/_meta/config.reference.yml > \
	     ${PREFIX}/${BEAT_NAME}-linux.reference.yml

# Collects all dependencies and then calls update
.PHONY: collect
collect: fields collect-docs configs kibana

# Collects all module and metricset fields
.PHONY: fields
fields: python-env
	@mkdir -p _meta
	@cp ${ES_BEATS}/metricbeat/_meta/fields.common.yml _meta/fields.generated.yml
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/metricbeat/scripts/fields_collector.py >> _meta/fields.generated.yml

# Collects all module configs
.PHONY: configs
configs: python-env
	@cat ${ES_BEATS}/auditbeat/_meta/common.p1.yml \
	     <(go run scripts/generate_config.go -os linux -concat) \
		 ${ES_BEATS}/auditbeat/_meta/common.p2.yml > _meta/beat.yml
	@cat ${ES_BEATS}/auditbeat/_meta/common.reference.yml \
	     <(go run scripts/generate_config.go -os linux -ref -concat) > _meta/beat.reference.yml

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
	@-rm -rf _meta/kibana
	@mkdir -p _meta/kibana
	@-cp -pr module/*/_meta/kibana _meta/

