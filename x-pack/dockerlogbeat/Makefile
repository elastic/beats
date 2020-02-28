# Name can be overwritten, as Metricbeat is also a library
BEAT_NAME?="Elastic logging plugin"
BEAT_TITLE?=dockerlogdriver
SYSTEM_TESTS?=false
TEST_ENVIRONMENT?=true
ES_BEATS?=../../

# Metricbeat can only be cross-compiled on platforms not requiring CGO.
GOX_OS=linux
GOX_FLAGS=-arch="amd64 386 arm ppc64 ppc64le"

include $(ES_BEATS)/dev-tools/make/xpack.mk


.PHONY: unit-tests
unit-tests: mage
	mage unitTest