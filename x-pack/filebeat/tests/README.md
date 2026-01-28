# Filebeat Integration Tests (X-Pack)

This directory contains integration tests for X-Pack Filebeat features.

For comprehensive documentation on Filebeat integration tests, including how to run tests, environment variables, and test structure, see:

**[Filebeat Integration Tests Documentation](../../../filebeat/tests/README.md)**

## X-Pack Specific Tests

The X-Pack Filebeat tests include:

- **`integration/`** - Go-based integration tests for X-Pack features
- **`system/`** - Python-based system tests for X-Pack modules and features
  - `test_xpack_modules.py` - Tests for X-Pack modules
  - `test_filebeat_xpack.py` - X-Pack Filebeat functionality tests
  - `test_http_endpoint.py` - HTTP endpoint tests

## Running X-Pack Module Tests

To run X-Pack module tests, set `MODULES_PATH` to point to the X-Pack modules directory:

```bash
cd x-pack/filebeat

INTEGRATION_TESTS=1 \
ES_USER="admin" \
ES_PASS="testing" \
MODULES_PATH="$(pwd)/module" \
pytest tests/system/test_xpack_modules.py
```

For documentation on module test environment variables, see:

**[Test_Modules.md](../../../filebeat/tests/Test_Modules.md)**
