BEAT_NAME=journalbeat
BEAT_TITLE=Journalbeat
SYSTEM_TESTS=false
TEST_ENVIRONMENT=false
ES_BEATS?=..

# Path to the libbeat Makefile
-include $(ES_BEATS)/libbeat/scripts/Makefile

.PHONY: before-build
before-build:

# Collects all dependencies and then calls update
.PHONY: collect
collect:
