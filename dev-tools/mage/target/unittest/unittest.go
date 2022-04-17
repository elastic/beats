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

package unittest

import (
	"context"

	"github.com/magefile/mage/mg"

	devtools "github.com/menderesk/beats/v7/dev-tools/mage"
	"github.com/menderesk/beats/v7/dev-tools/mage/target/test"
)

func init() {
	test.RegisterDeps(UnitTest)
}

var (
	goTestDeps, pythonTestDeps []interface{}
)

// RegisterGoTestDeps registers dependencies of the GoUnitTest target.
func RegisterGoTestDeps(deps ...interface{}) {
	goTestDeps = append(goTestDeps, deps...)
}

// RegisterPythonTestDeps registers dependencies of the PythonUnitTest target.
func RegisterPythonTestDeps(deps ...interface{}) {
	pythonTestDeps = append(pythonTestDeps, deps...)
}

// UnitTest executes the unit tests (Go and Python).
func UnitTest() {
	mg.SerialDeps(GoUnitTest, PythonUnitTest)
}

// GoUnitTest executes the Go unit tests.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
func GoUnitTest(ctx context.Context) error {
	mg.SerialCtxDeps(ctx, goTestDeps...)
	return devtools.GoTest(ctx, devtools.DefaultGoTestUnitArgs())
}

// PythonUnitTest executes the python system tests.
func PythonUnitTest() error {
	mg.SerialDeps(pythonTestDeps...)
	mg.Deps(devtools.BuildSystemTestBinary)
	return devtools.PythonTest(devtools.DefaultPythonTestUnitArgs())
}
