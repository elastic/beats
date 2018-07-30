BEAT_NAME=heartbeat
BEAT_TITLE=Heartbeat
SYSTEM_TESTS=true
TEST_ENVIRONMENT=false

# Path to the libbeat Makefile
-include ../libbeat/scripts/Makefile

# Collects all dependencies and then calls update
.PHONY: collect
collect: imports kibana

# Generate imports for all monitors
.PHONY: imports
imports: python-env
	@mkdir -p include
	@${PYTHON_ENV}/bin/python ${ES_BEATS}/script/generate_imports.py --out monitors/defaults/default.go ${BEAT_PATH}

# Create a new monitor. Requires the parameter MONITOR
.PHONY: create-monitor
create-monitor:
	go run scripts/generate_monitor/main.go -monitor=${MONITOR}

# Collects all module dashboards
.PHONY: kibana
kibana:
	@rm -rf _meta/kibana.generated
	@mkdir -p _meta/kibana.generated
	@-cp -r monitors/active/*/_meta/kibana/* _meta/kibana.generated
