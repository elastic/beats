// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build mage

package main

import (
	"context"
	"fmt"

	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/elastic/beats/v7/dev-tools/mage/target/test"

	//mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/common"
	//mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/build"
	//mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/pkg"
	//mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/dashboards"
	//mage:import
	winlogbeat "github.com/elastic/beats/v7/winlogbeat/scripts/mage"
)

func init() {
	winlogbeat.SelectLogic = devtools.XPackProject
	devtools.BeatLicense = "Elastic License"

	RegisterGoTestDeps(winlogbeat.Update.Fields)
	test.RegisterDeps(UnitTest)
}

var goTestDeps, pythonTestDeps []interface{}

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

// Update is an alias for update:all. This is a workaround for
// https://github.com/magefile/mage/issues/217.
func Update() { mg.Deps(winlogbeat.Update.All) }

// GoUnitTest executes the Go unit tests.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
func GoUnitTest(ctx context.Context) error {
	mg.SerialCtxDeps(ctx, goTestDeps...)
	args := devtools.DefaultGoTestUnitArgs()
	// The module unit tests depend on a running docker container to provide
	// the ES instance to run the processor pipeline. In the absence of a
	// test supervisor or a single test executable to ensure that only a
	// single container is running, or additional logic to ensure no network
	// collisions, we ensure that only one test package is running at a time.
	args.ExtraFlags = append(args.ExtraFlags, "-p", "1")
	return devtools.GoTest(ctx, args)
}

// PythonUnitTest executes the python system tests.
func PythonUnitTest() error {
	mg.SerialDeps(pythonTestDeps...)
	mg.Deps(devtools.BuildSystemTestBinary)
	return devtools.PythonTest(devtools.DefaultPythonTestUnitArgs())
}

// PythonVirtualEnv creates the testing virtual environment and prints its location.
func PythonVirtualEnv() error {
	venv, err := devtools.PythonVirtualenv(true)
	if err != nil {
		return err
	}
	fmt.Println(venv)
	return nil
}

// Package packages the Beat for IronBank distribution.
//
// Use SNAPSHOT=true to build snapshots.
func Ironbank() error {
	fmt.Println(">> Ironbank: this module is not subscribed to the IronBank releases.")
	return nil
}
