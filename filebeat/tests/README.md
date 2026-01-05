# Filebeat Integration Tests

This directory contains integration tests for Filebeat. Integration tests verify that Filebeat works correctly with external systems like Elasticsearch, Kibana, and various log sources.

## Test Structure

The Filebeat test suite is organized into several directories:

- **`integration/`** - Go-based integration tests that test Filebeat functionality directly
- **`system/`** - Python-based system tests that run Filebeat as a complete system
- **`load/`** - Load testing utilities, deprecated
- **`files/`** - Test data files (logs, configs, registry files), deprecated

## Test Types

### Go Integration Tests (`integration/`)

Go integration tests are unit tests that can run with external services. They use the `//go:build integration` build tag and have the `_integration_test.go` suffix.

**Running Go Integration Tests:**

```bash
cd filebeat

# Run all integration tests (starts required services automatically)
mage goIntegTest

# Run specific integration tests
go test ./filebeat/tests/integration/... -tags integration

# Run with race detector
RACE_DETECTOR=1 mage goIntegTest
```

### Python System Tests (`system/`)

Python system tests run Filebeat as a complete system, testing end-to-end functionality including:
- Module processing and pipelines
- Input/output configurations
- Autodiscover functionality
- Reload behavior
- Various input types (file, TCP, UDP, syslog, etc.)

**Running Python System Tests:**

```bash
cd filebeat

# Set up Python environment (first time only)
source $(mage pythonVirtualEnv)/bin/activate

# Build test binary
mage buildSystemTestBinary

# Run all system tests (starts required services automatically)
mage pythonIntegTest

# Run specific test file
INTEGRATION_TESTS=1 \
ES_USER="admin" \
ES_PASS="testing" \
pytest tests/system/test_modules.py

# Run with specific environment variables
INTEGRATION_TESTS=1 \
BEAT_STRICT_PERMS=false \
ES_USER="admin" \
ES_PASS="testing" \
pytest tests/system/test_modules.py
```

## Environment Variables

### Required for System Tests

- **`INTEGRATION_TESTS`** - Must be set to `1` to enable integration tests. Without this, tests are skipped.

### Common Environment Variables

- **`BEAT_STRICT_PERMS`** - Set to `false` to disable strict permission checks
- **`ES_USER`** - Elasticsearch username (default: `admin`)
- **`ES_PASS`** - Elasticsearch password (default: `testing`)
- **`TESTING_ENVIRONMENT`** - Set to `"2x"` to skip tests incompatible with ES 2.x

### Module Test Variables

For detailed documentation on environment variables that control module tests, see:

- **[Test_Modules.md](./Test_Modules.md)** - Complete documentation of all environment variables for `test_modules.py`

Key variables include:
- `TESTING_FILEBEAT_MODULES` - Limit tests to specific modules (comma-separated)
- `TESTING_FILEBEAT_FILESETS` - Limit tests to specific filesets (comma-separated)
- `TESTING_FILEBEAT_FILEPATTERN` - Control which file patterns are tested
- `MODULES_PATH` - Override the modules directory path
- `GENERATE` - Generate expected JSON files instead of comparing

## Manual Test Execution

To run tests manually with full control over services:

```bash
cd filebeat

# Set up Python environment
source $(mage pythonVirtualEnv)/bin/activate

# Build and start Docker services
mage docker:composeUp

# Build test binary
mage buildSystemTestBinary

# Run tests
INTEGRATION_TESTS=1 \
BEAT_STRICT_PERMS=false \
ES_USER="admin" \
ES_PASS="testing" \
pytest tests/system

# Stop services when done
mage docker:composeDown
```

## Running Specific Tests

### Run tests for a single module

```bash
INTEGRATION_TESTS=1 \
ES_USER="admin" \
ES_PASS="testing" \
TESTING_FILEBEAT_MODULES=apache \
pytest tests/system/test_modules.py
```

### Run tests for specific filesets

```bash
INTEGRATION_TESTS=1 \
ES_USER="admin" \
ES_PASS="testing" \
TESTING_FILEBEAT_MODULES=nginx \
TESTING_FILEBEAT_FILESETS=access,error \
pytest tests/system/test_modules.py
```

### Run a specific test file

```bash
INTEGRATION_TESTS=1 \
ES_USER="admin" \
ES_PASS="testing" \
pytest tests/system/test_input.py
```

### Run a specific test function

```bash

INTEGRATION_TESTS=1 ES_PASS=testing ES_USER=admin pytest tests/system/test_setup.py::Test::test_setup_modules_d_config
```

## Debugging Tests

### View Filebeat logs

Test output and Filebeat logs are written to the working directory (typically `tests/system/`). Check `output.log` files for detailed execution logs.

### Generate expected files

When updating module tests, you can regenerate expected JSON files:

```bash
INTEGRATION_TESTS=1 \
ES_USER="admin" \
ES_PASS="testing" \
TESTING_FILEBEAT_MODULES=apache \
GENERATE=1 \
pytest tests/system/test_modules.py
```

### Skip diff comparison

When testing against older Elasticsearch versions that produce slightly different documents:

```bash
INTEGRATION_TESTS=1 \
ES_USER="admin" \
ES_PASS="testing" \
TESTING_FILEBEAT_ALLOW_OLDER=1 \
TESTING_FILEBEAT_SKIP_DIFF=1 \
pytest tests/system/test_modules.py
```

## Additional Resources

- [Beats Testing Guide](../../docs/extend/testing.md) - General testing documentation
- [Filebeat Module Development Guide](../../docs/extend/filebeat-modules-devguide.md) - Module development and testing
- [Test_Modules.md](./Test_Modules.md) - Detailed module test environment variables

## CI/CD

Tests are automatically run in CI/CD pipelines. To run the same test suite locally:

```bash
cd filebeat
mage pythonIntegTest  # Python system tests
mage goIntegTest      # Go integration tests
```
