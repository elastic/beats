ES_BEATS ?= ../..

# This version is used to build the OpenTelemetry Collector distribution.
# It should be updated when the OTel Collector version is updated in the beats-otel-collector module.
OCB_VERSION ?= v0.155.0

include $(ES_BEATS)/dev-tools/make/mage.mk

.PHONY: build-otel-distro-ci
build-otel-distro-ci:
	go install go.opentelemetry.io/collector/cmd/builder@$(OCB_VERSION)
	mv $$(go env GOPATH)/bin/builder $$(go env GOPATH)/bin/ocb
	PATH="$$(go env GOPATH)/bin:$$PATH" RACE=1 mage buildOtelDistro
	go version -m ./distributions/beats-otel-collector/_build/beats-otel-collector | grep -q 'race=true'
	timeout 30s mage runOtelDistro | tee /tmp/collector.log 2>&1 || true
	echo "--- Collector output (errors below are expected if external services are unavailable)"
	cat /tmp/collector.log
	echo "---"
	grep -q "Starting beats-otel-collector" /tmp/collector.log
