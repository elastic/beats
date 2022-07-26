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
	"github.com/magefile/mage/sh"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	heartbeat "github.com/elastic/beats/v7/heartbeat/scripts/mage"

	// mage:import
	"github.com/elastic/beats/v7/dev-tools/mage/target/common"
	// mage:import
	"github.com/elastic/beats/v7/dev-tools/mage/target/build"

	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/unittest"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/integtest/notests"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/test"
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

// ValidateIronbank validates the existing dependencies needed
// for the Ironbank have not changed and if so, then fail the build.
//
func ValidateIronbank() error {
	start := time.Now()
	defer func() { fmt.Println("validateIronbank ran for", time.Since(start)) }()
	// TODO: generate dependencies file (rpm-deps.txt) and compare with
	return customiseIronbank()
}

// Ironbank packages the Beat for IronBank distribution.
//
// Use SNAPSHOT=true to build snapshots.
func Ironbank() error {
	start := time.Now()
	defer func() { fmt.Println("ironbank ran for", time.Since(start)) }()
	err := customiseIronbank()
	if err != nil {
		return err
	}
	return devtools.Ironbank()
}

func customiseIronbank() error {
	fmt.Println(">>> customiseIronbank (I'm downloading all the required dependencies...)")
	makeIronbankPrepare := sh.OutCmd("make", "-C", "ironbank", "prepare")
	if out, err := makeIronbankPrepare(); err != nil {
		fmt.Println(out)
		return err
	}
	return nil
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
