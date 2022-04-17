// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package integtest

import (
	"context"

	"github.com/magefile/mage/mg"

	devtools "github.com/menderesk/beats/v7/dev-tools/mage"
	"github.com/menderesk/beats/v7/dev-tools/mage/target/test"
)

func init() {
	test.RegisterDeps(IntegTest)
}

var (
	goTestDeps, pythonTestDeps []interface{}
	whitelistedEnvVars         []string
)

// RegisterGoTestDeps registers dependencies of the GoIntegTest target.
func RegisterGoTestDeps(deps ...interface{}) {
	goTestDeps = append(goTestDeps, deps...)
}

// RegisterPythonTestDeps registers dependencies of the PythonIntegTest target.
func RegisterPythonTestDeps(deps ...interface{}) {
	pythonTestDeps = append(pythonTestDeps, deps...)
}

// WhitelistEnvVar whitelists an environment variable to enabled it to be
// passed into the clean integration test environment (Docker).
func WhitelistEnvVar(key ...string) {
	whitelistedEnvVars = append(whitelistedEnvVars, key...)
}

// IntegTest executes integration tests (it uses Docker to run the tests).
func IntegTest() {
	mg.SerialDeps(GoIntegTest, PythonIntegTest)
}

// GoIntegTest executes the Go integration tests.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
func GoIntegTest(ctx context.Context) error {
	if !devtools.IsInIntegTestEnv() {
		mg.SerialDeps(goTestDeps...)
	}
	runner, err := devtools.NewDockerIntegrationRunner(whitelistedEnvVars...)
	if err != nil {
		return err
	}
	return runner.Test("goIntegTest", func() error {
		return devtools.GoTest(ctx, devtools.DefaultGoTestIntegrationArgs())
	})
}

// PythonIntegTest executes the python system tests in the integration
// environment (Docker).
// Use PYTEST_ADDOPTS="-k pattern" to only run tests matching the specified pattern.
// Use any other PYTEST_* environment variable to influence the behavior of pytest.
func PythonIntegTest(ctx context.Context) error {
	if !devtools.IsInIntegTestEnv() {
		mg.SerialDeps(pythonTestDeps...)
	}
	runner, err := devtools.NewDockerIntegrationRunner(append(whitelistedEnvVars, devtools.ListMatchingEnvVars("PYTEST_")...)...)
	if err != nil {
		return err
	}
	return runner.Test("pythonIntegTest", func() error {
		mg.Deps(devtools.BuildSystemTestBinary)
		return devtools.PythonTest(devtools.DefaultPythonTestIntegrationArgs())
	})
}
