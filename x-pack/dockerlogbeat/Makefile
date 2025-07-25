# Name can be overwritten, as Metricbeat is also a library
BEAT_NAME?="Elastic logging plugin"
BEAT_TITLE?=dockerlogdriver
SYSTEM_TESTS?=false
TEST_ENVIRONMENT?=true
ES_BEATS?=../../

include $(ES_BEATS)/dev-tools/make/mage.mk


.PHONY: unit-tests
unit-tests: mage
	mage unitTest
