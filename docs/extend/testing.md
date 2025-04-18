---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/testing.html
---

# Testing [testing]

Beats has a various sets of tests. This guide should help to understand how the different test suites work, how they are used and new tests are added.

In general there are two major test suites:

* Tests written in Go
* Tests written in Python

The tests written in Go use the [Go Testing package](https://golang.org/pkg/testing/). The tests written in Python depend on [pytest](https://docs.pytest.org/en/latest/) and require a compiled and executable binary from the Go code. The python test run a beat with a specific config and params and either check if the output is as expected or if the correct things show up in the logs.

For both of the above test suites so called integration tests exists. Integration tests in Beats are tests which require an external system like Elasticsearch to test if the integration with this service works as expected. Beats provides in its testsuite docker containers and docker-compose files to start these environments but a developer can run the required services also locally.

## Running Go Tests [_running_go_tests]

The Go tests can be executed in each Go package by running `go test .`. This will execute all tests which don’t don’t require an external service to be running. To run all non integration tests for a beat run `mage unitTest`.

All Go tests are in the same package as the tested code itself and have the suffix `_test` in the file name. Most of the tests are in the same package as the rest of the code. Some of the tests which should be separate from the rest of the code or should not use private variables go under `{{packagename}}_test`.

### Running Go Integration Tests [_running_go_integration_tests]

Integration tests are labelled with the `//go:build integration` build tag and use the `_integration_test.go` suffix.

To run the integration tests use the `mage goIntegTest` target, which will start the required services using [docker-compose](https://docs.docker.com/compose/) and run all integration tests.

It is also possible to run module specific integration tests. For example, to run kafka only tests use `MODULE=kafka mage integTest -v`

It is possible to start the `docker-compose` services manually to allow selecting which specific tests should be run. The default credentials for Elasticsearch and Kibana are `user: admin`, `password: testing`. An example follows for filebeat:

```bash
cd filebeat
# Pull and build the containers. Only needs to be done once unless you change the containers.
mage docker:composeBuild

# Bring up all containers, wait until they are healthy, and put them in the background.
mage docker:composeUp

# Build the test binary
mage buildSystemTestBinary

# Run all integration tests.
go test ./filebeat/...  -tags integration

# Stop all started containers.
mage docker:composeDown
```


### Generate sample events [_generate_sample_events]

Go tests support generating sample events to be used as fixtures.

This generation can be perfomed running `go test --data`. This functionality is supported by packetbeat and Metricbeat.

In Metricbeat, run the command from within a module like this: `go test --tags integration,azure --data --run "TestData"`. Make sure to add the relevant tags (`integration` is common then add module and metricset specific tags).

A note about tags: the `--data` flag is a custom flag added by Metricbeat and Packetbeat frameworks. It will not be present in case tags do not match, as the relevant code will not be run and silently skipped (without the tag the test file is ignored by Go compiler so the framework doesn’t load). This may happen if there are different tags in the build tags of the metricset under test (i.e. the GCP billing metricset requires the `billing` tag too).



## Running System (integration) Tests (Python and Go) [_running_system_integration_tests_python_and_go]

The system tests are defined in the `tests/system` (for legacy Python test) and on `tests/integration` (for Go tests) directory. They require a testing binary to be available and the python environment to be set up.

To create the testing binary run `mage buildSystemTestBinary`. This will create the test binary in the beat directory. To set up the Python testing environment run `mage pythonVirtualEnv` which will create a virtual environment with all test dependencies and print its location. To activate it, the instructions depend on your operating system. See the [virtualenv documentation](https://packaging.python.org/en/latest/guides/installing-using-pip-and-virtual-environments/#activating-a-virtual-environment).

To run the system and integration tests use the `mage pythonIntegTest` target, which will start the required services using [docker-compose](https://docs.docker.com/compose/) and run all integration tests. The default credentials for Elasticsearch and Kibana are `user: admin`, `password: testing`. Similar to Go integration tests, the individual steps can be done manually to allow selecting which tests should be run:

```bash
# Create and activate the system test virtual environment (assumes a Unix system).
source $(mage pythonVirtualEnv)/bin/activate

# Pull and build the containers. Only needs to be done once unless you change the containers.
mage docker:composeBuild

# Bring up all containers, wait until they are healthy, and put them in the background.
mage docker:composeUp

# Build the test binary
mage buildSystemTestBinary

# Run all system and integration tests.
INTEGRATION_TESTS=1 \
BEAT_STRICT_PERMS=false \
ES_USER="admin" \
ES_PASS="testing" \
pytest ./tests/system

# Stop all started containers.
mage docker:composeDown
```

To run a module test from `x-pack/`, you need to set the `MODULES_PATH` to the full
path to the `x-pack` modules directory:

```bash
cd x-pack/filebeat

INTEGRATION_TESTS=1 \
BEAT_STRICT_PERMS=false \
ES_USER="admin" \
ES_PASS="testing" \
MODULES_PATH="/path/to/beats-repo/x-pack/filebeat/module" \
pytest ./tests/system/test_xpack_modules.py
```

Filebeat’s module python tests have additional documentation found in the [Filebeat module](/extend/filebeat-modules-devguide.md) guide.


## Test commands [_test_commands]

To list all mage commands run `mage -l`. A quick summary of the available test Make commands is:

* `unit`: Go tests
* `unit-tests`: Go tests with coverage reports
* `integration-tests`: Go tests with services in local docker
* `integration-tests-environment`: Go tests inside docker with service in docker
* `fast-system-tests`: Python tests
* `system-tests`: Python tests with coverage report
* `INTEGRATION_TESTS=1 system-tests`: Python tests with local services
* `system-tests-environment`: Python tests inside docker with service in docker
* `testsuite`: Complete test suite in docker environment is run
* `test`: Runs testsuite without environment

There are two experimental test commands:

* `benchmark-tests`: Running Go tests with `-bench` flag
* `load-tests`: Running system tests with `LOAD_TESTS=1` flag


## Coverage report [_coverage_report]

If the tests were run to create a test coverage, the coverage report files can be found under `build/docs`. To create a more human readable file out of the `.cov` file `make coverage-report` can be used. It creates a `.html` file for each report and a `full.html` as summary of all reports together in the directory `build/coverage`.


## Race detection [_race_detection]

All tests can be run with the Go race detector enabled by setting the environment variable `RACE_DETECTOR=1`. This applies to tests in Go and Python. For Python the test binary has to be recompile when the flag is changed. Having the race detection enabled will slow down the tests.

## Stress testing [_stress_testing]

Stress testing can be useful to reproduce CI flaky runs or check for flakiness in tests.

```bash
./script/stresstest.sh --help
./script/stresstest.sh ./libbeat/common/backoff ^TestBackoff$ -p 32
```
