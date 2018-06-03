BEAT_NAME=libbeat
TEST_ENVIRONMENT?=true
SYSTEM_TESTS=true
FIELDS_FILE_PATH=processors

include scripts/Makefile

# Collects all dependencies and then calls update
.PHONY: collect
collect: libbeat_fields
