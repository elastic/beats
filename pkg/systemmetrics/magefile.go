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

//go:build mage

package main

import (
	"context"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"

	// mage:import provides the shared "check" and "fmt" targets (and a few
	// others) used across all Beats, so pkg/systemmetrics stays in lockstep
	// with the rest of the repo.
	//mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/common"
)

// Build compiles all Go packages in pkg/systemmetrics. Unlike a Beat,
// systemmetrics is a library with no main package, so there is no binary to
// produce; we simply ensure every package compiles.
func Build() error {
	return sh.RunV("go", "build", "./...")
}

// UnitTest executes the unit tests.
func UnitTest() {
	mg.SerialDeps(GoUnitTest)
}

// GoUnitTest executes the Go unit tests.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
func GoUnitTest(ctx context.Context) error {
	return devtools.GoTest(ctx, devtools.DefaultGoTestUnitArgs())
}

// IntegTest executes the integration tests.
func IntegTest() {
	mg.SerialDeps(GoIntegTest)
}

// GoIntegTest executes the Go integration (container-system) tests. These tests
// bring up their own Docker containers via the Docker client and only require a
// running Docker daemon, so they do not need the docker-compose integration
// stack that Beats normally use.
func GoIntegTest(ctx context.Context) error {
	args := devtools.DefaultGoTestIntegrationArgs(ctx)
	args.Packages = []string{"./tests/..."}
	// The container tests spin up their own Docker containers and can run for
	// up to ~20 minutes (e.g. TestProcessAllSettings sleeps for 1200s), so use a
	// generous timeout.
	args.ExtraFlags = []string{"-count=1", "-timeout=30m"}
	return devtools.GoTest(ctx, args)
}
