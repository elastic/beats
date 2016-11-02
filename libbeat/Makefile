
BEATNAME=libbeat
TEST_ENVIRONMENT=true
SYSTEM_TESTS=true

include scripts/Makefile

# Collects all fields from processors
.PHONY: fields
fields:
	cat _meta/fields_base.yml > _meta/fields.yml
	cat processors/*/_meta/fields.yml >> _meta/fields.yml

# Collects all dependencies and then calls update
.PHONY: collect
collect: fields
