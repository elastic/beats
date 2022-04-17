// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/magefile/mage/mg"

	devtools "github.com/menderesk/beats/v7/dev-tools/mage"
	heartbeat "github.com/menderesk/beats/v7/heartbeat/scripts/mage"

	// mage:import
	"github.com/menderesk/beats/v7/dev-tools/mage/target/common"
	// mage:import
	"github.com/menderesk/beats/v7/dev-tools/mage/target/build"

	// mage:import
	_ "github.com/menderesk/beats/v7/dev-tools/mage/target/unittest"
	// mage:import
	_ "github.com/menderesk/beats/v7/dev-tools/mage/target/integtest/notests"
	// mage:import
	_ "github.com/menderesk/beats/v7/dev-tools/mage/target/test"
)

func init() {
	common.RegisterCheckDeps(Update)

	devtools.BeatLicense = "Elastic License"
}

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
// Use VERSION_QUALIFIER to control the version qualifier.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	if v, found := os.LookupEnv("AGENT_PACKAGING"); found && v != "" {
		devtools.UseElasticBeatXPackReducedPackaging()
	} else {
		devtools.UseElasticBeatXPackPackaging()
	}

	devtools.PackageKibanaDashboardsFromBuildDir()
	heartbeat.CustomizePackaging()

	mg.Deps(Update)
	mg.Deps(build.CrossBuild, build.CrossBuildGoDaemon)
	mg.SerialDeps(devtools.Package, TestPackages)
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return devtools.TestPackages(devtools.WithMonitorsD())
}

// Update updates the generated files (aka make update).
func Update() {
	mg.SerialDeps(Fields, FieldDocs, Config)
}

func Fields() error {
	return heartbeat.Fields()
}

func FieldDocs() error {
	return devtools.Docs.FieldDocs("fields.yml")
}

// Config generates both the short/reference/docker configs.
func Config() error {
	return devtools.Config(devtools.AllConfigTypes, heartbeat.ConfigFileParams(), ".")
}
