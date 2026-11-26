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
	"fmt"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"

	// mage:import provides the shared "check" and "fmt" targets (and a few
	// others) used across all Beats, so pkg stays in lockstep with the rest
	// of the repo.
	//mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/common"
)

const systemMetricsPkg = "github.com/elastic/beats/v7/pkg/systemmetrics"

// Build compiles all Go packages in pkg. These are libraries with no main
// package, so there is no binary to produce; we simply ensure every package
// compiles.
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
// Use FIPS=true to add the requirefips build tag.
func GoUnitTest(ctx context.Context) error {
	return devtools.GoTest(ctx, devtools.DefaultGoTestUnitArgs())
}

// GoFIPSOnlyUnitTest executes the Go unit tests with GODEBUG=fips140=only.
// Use FIPS=true to add the requirefips build tag.
func GoFIPSOnlyUnitTest(ctx context.Context) error {
	return devtools.GoTest(ctx, devtools.DefaultGoFIPSOnlyTestArgs())
}

// IntegTest executes the integration tests.
func IntegTest() {
	mg.SerialDeps(GoIntegTest, GoSystemMetricsContainerTest)
}

// GoIntegTest executes the Go tests built with the integration tag for every
// package except systemmetrics, whose integration tests need a dedicated
// container setup (see GoSystemMetricsContainerTest). These tests only need
// a local Docker daemon, not the docker-compose integration stack that Beats
// normally use.
func GoIntegTest(ctx context.Context) error {
	pkgs, err := packagesExcluding(systemMetricsPkg)
	if err != nil {
		return err
	}
	args := devtools.DefaultGoTestIntegrationArgs(ctx)
	args.Packages = pkgs
	return devtools.GoTest(ctx, args)
}

// GoSystemMetricsContainerTest executes the systemmetrics container-system
// tests. They bring up their own Docker containers via the Docker client and
// only require a running Docker daemon.
func GoSystemMetricsContainerTest(ctx context.Context) error {
	args := devtools.DefaultGoTestIntegrationArgs(ctx)
	args.TestName = "SystemMetrics-Container"
	args.OutputFile = "build/TEST-go-systemmetrics-container.out"
	args.JUnitReportFile = "build/TEST-go-systemmetrics-container.xml"
	if args.CoverageProfileFile != "" {
		args.CoverageProfileFile = "build/TEST-go-systemmetrics-container.cov"
	}
	args.Packages = []string{"./systemmetrics/tests/..."}
	// The container tests can run for up to ~20 minutes (e.g.
	// TestProcessAllSettings sleeps for 1200s), so use a generous timeout.
	args.ExtraFlags = []string{"-count=1", "-timeout=30m"}
	return devtools.GoTest(ctx, args)
}

// packagesExcluding lists the packages under the current directory, leaving
// out the excluded package and everything below it.
func packagesExcluding(excluded string) ([]string, error) {
	out, err := sh.Output("go", "list", "-tags", "integration", "./...")
	if err != nil {
		return nil, fmt.Errorf("listing packages: %w", err)
	}
	var pkgs []string
	for pkg := range strings.FieldsSeq(out) {
		if pkg == excluded || strings.HasPrefix(pkg, excluded+"/") {
			continue
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}
