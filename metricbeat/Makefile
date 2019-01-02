#
# Variables
#
ES_BEATS  ?= ..
GOX_OS    ?= netbsd linux windows
GOX_FLAGS ?= -arch="amd64 386 arm ppc64 ppc64le"

#
# Includes
#
include $(ES_BEATS)/dev-tools/make/oss.mk

#
# Targets
#

# Creates a new metricset. Requires the params MODULE and METRICSET.
.PHONY: create-metricset
create-metricset: mage
	mage generateMetricSet

#
# TODO (andrewkroh on 12-26-2018): These targets need refactored to work in the
# Mage based build system.
#

# Generates the data.json example documents.
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
	. ${PYTHON_VE_DIR}/bin/activate && INTEGRATION_TESTS=1 nosetests tests/system/test_${MODULE}.py
