BEATNAME=heartbeat
BEAT_DESCRIPTION?=Ping remote services for availablity and log results to Elasticsearch or send to Logstash.
SYSTEM_TESTS=false
TEST_ENVIRONMENT=false

# Path to the libbeat Makefile
-include ../libbeat/scripts/Makefile

# This is called by the beats packer before building starts
.PHONY: before-build
before-build:

# Collects all dependencies and then calls update
.PHONY: collect
collect:
