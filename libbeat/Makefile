
BEATNAME=libbeat
TEST_ENVIRONMENT=true
SYSTEM_TESTS=true

include scripts/Makefile

# Collects all fields from processors
.PHONY: fields
fields:
	cat _meta/fields.common.yml > _meta/fields.generated.yml
	cat processors/*/_meta/fields.yml >> _meta/fields.generated.yml

# Collects all dependencies and then calls update
.PHONY: collect
collect: fields
