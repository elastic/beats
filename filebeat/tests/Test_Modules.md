# Environment Variables for test_modules.py

This document describes all environment variables that control which tests run in `filebeat/tests/system/test_modules.py`.

## Summary

Found 9 environment variables:

**Test Selection Variables:**
1. `TESTING_FILEBEAT_MODULES` - Limits tests to specific modules (comma-separated)
2. `TESTING_FILEBEAT_FILESETS` - Limits tests to specific filesets (comma-separated)
3. `TESTING_FILEBEAT_FILEPATTERN` - Controls which file patterns are tested (default: `*.log,*.journal`)
4. `MODULES_PATH` - Overrides the modules directory path

**Test Execution Control:**
5. `INTEGRATION_TESTS` - Must be set to `1` to enable tests (default: tests skipped)
6. `TESTING_ENVIRONMENT` - When set to `"2x"`, skips all tests

**Test Behavior:**
7. `TESTING_FILEBEAT_ALLOW_OLDER` - Allows connecting to older Elasticsearch versions
8. `TESTING_FILEBEAT_SKIP_DIFF` - Skips diff comparison (useful for older ES versions)
9. `GENERATE` - Generates expected JSON files instead of comparing

## Usage Examples

### Run tests for a single module
```bash
INTEGRATION_TESTS=1 TESTING_FILEBEAT_MODULES=apache pytest filebeat/tests/system/test_modules.py
```

### Run tests for specific filesets in a module
```bash
INTEGRATION_TESTS=1 TESTING_FILEBEAT_MODULES=nginx TESTING_FILEBEAT_FILESETS=access,error pytest filebeat/tests/system/test_modules.py
```

### Run tests only for .log files (exclude .journal)
```bash
INTEGRATION_TESTS=1 TESTING_FILEBEAT_FILEPATTERN=*.log pytest filebeat/tests/system/test_modules.py
```

### Generate expected files for a module
```bash
INTEGRATION_TESTS=1 TESTING_FILEBEAT_MODULES=apache GENERATE=1 pytest filebeat/tests/system/test_modules.py
```

### Test against older Elasticsearch version
```bash
INTEGRATION_TESTS=1 TESTING_FILEBEAT_ALLOW_OLDER=1 TESTING_FILEBEAT_SKIP_DIFF=1 pytest filebeat/tests/system/test_modules.py
```

## Test Selection Variables

### `TESTING_FILEBEAT_MODULES`
- **Type**: Comma-separated string
- **Default**: All modules in the modules directory
- **Usage**: Limits tests to specific modules
- **Example**: `TESTING_FILEBEAT_MODULES=apache,nginx`
- **Location**: `filebeat/tests/system/test_modules.py:82-86`
- **Description**: When set, only tests for the specified modules will run. If not set, all modules in the modules directory are tested.

### `TESTING_FILEBEAT_FILESETS`
- **Type**: Comma-separated string
- **Default**: All filesets in each module directory
- **Usage**: Limits tests to specific filesets within modules
- **Example**: `TESTING_FILEBEAT_FILESETS=access,error`
- **Location**: `filebeat/tests/system/test_modules.py:88-101`
- **Description**: When set, only tests for the specified filesets will run. If not set, all filesets in each module are tested. Works in combination with `TESTING_FILEBEAT_MODULES`.

### `TESTING_FILEBEAT_FILEPATTERN`
- **Type**: Comma-separated string
- **Default**: `"*.log,*.journal"`
- **Usage**: Controls which file patterns/extensions are tested
- **Example**: `TESTING_FILEBEAT_FILEPATTERN=*.log,*.txt`
- **Location**: `filebeat/tests/system/test_modules.py:110`
- **Description**: Specifies which file patterns to search for in the test directories. Files matching these patterns are used as test inputs.

### `MODULES_PATH`
- **Type**: String (directory path)
- **Default**: `{test_file_dir}/../../module` (relative to test file)
- **Usage**: Overrides the default modules directory path
- **Example**: `MODULES_PATH=/custom/path/to/modules`
- **Location**: `filebeat/tests/system/test_modules.py:78-81`
- **Description**: Sets the base directory where modules are located. If not set, defaults to the standard module directory relative to the test file location.

## Test Execution Control Variables

### `INTEGRATION_TESTS`
- **Type**: String (truthy value, typically `"1"`)
- **Default**: `False` (tests are skipped)
- **Usage**: Enables/disables integration tests
- **Example**: `INTEGRATION_TESTS=1`
- **Location**: `filebeat/tests/system/test_modules.py:137-138` (imported from `beat.beat`)
- **Description**: When not set or falsy, all tests are skipped with message "integration tests are disabled, run with INTEGRATION_TESTS=1 to enable them." Tests only run when this is set to a truthy value (typically `"1"`).

### `TESTING_ENVIRONMENT`
- **Type**: String
- **Default**: None
- **Usage**: Skips tests for specific environments
- **Example**: `TESTING_ENVIRONMENT=2x`
- **Location**: `filebeat/tests/system/test_modules.py:139-140`
- **Description**: When set to `"2x"`, all tests are skipped with message "integration test not available on 2.x". Used to disable tests for older Elasticsearch versions.

## Test Behavior Variables

### `TESTING_FILEBEAT_ALLOW_OLDER`
- **Type**: String (any truthy value)
- **Default**: Not set
- **Usage**: Allows connecting to older versions of Elasticsearch
- **Example**: `TESTING_FILEBEAT_ALLOW_OLDER=1`
- **Location**: `filebeat/tests/system/test_modules.py:198-199`
- **Description**: When set, adds `-E output.elasticsearch.allow_older_versions=true` to the Filebeat command, allowing connections to older Elasticsearch versions that might otherwise be rejected.

### `TESTING_FILEBEAT_SKIP_DIFF`
- **Type**: String (any truthy value)
- **Default**: Not set
- **Usage**: Skips comparison between expected and actual results
- **Example**: `TESTING_FILEBEAT_SKIP_DIFF=1`
- **Location**: `filebeat/tests/system/test_modules.py:322-323`
- **Description**: When set, skips the DeepDiff comparison between expected and actual documents. Useful when testing against older ES versions that produce slightly different documents, avoiding the need to maintain multiple sets of golden files.

### `GENERATE`
- **Type**: String (any truthy value)
- **Default**: Not set
- **Usage**: Generates expected JSON files instead of comparing against them
- **Example**: `GENERATE=1`
- **Location**: `filebeat/tests/system/test_modules.py:296-308`
- **Description**: When set, generates `{test_file}-expected.json` files from the actual test results. The generated files are cleaned and normalized (flattened, sorted, keys cleaned) to ensure consistency across different machines/versions.
