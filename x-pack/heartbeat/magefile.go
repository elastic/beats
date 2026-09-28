// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build mage

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/elastic/beats/v7/dev-tools/mage/target/test"
	heartbeat "github.com/elastic/beats/v7/heartbeat/scripts/mage"

	// mage:import
	"github.com/elastic/beats/v7/dev-tools/mage/target/common"
	// mage:import
	"github.com/elastic/beats/v7/dev-tools/mage/target/build"

	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/unittest"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/integtest/docker"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/test"
)

func init() {
	common.RegisterCheckDeps(Update)
	test.RegisterDeps(IntegTest)

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
	mg.Deps(build.CrossBuild)
	mg.SerialDeps(devtools.Package, TestPackages)
}

// Ironbank packages the Beat for IronBank distribution.
//
// Use SNAPSHOT=true to build snapshots.
func Ironbank() error {
	start := time.Now()
	defer func() { fmt.Println("ironbank ran for", time.Since(start)) }()
	return devtools.Ironbank()
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return devtools.TestPackages(devtools.WithMonitorsD())
}

func GenerateModuleIncludeListGo() error {
	opts := devtools.DefaultIncludeListOptions()
	opts.ImportDirs = append(opts.ImportDirs, "monitors/*")
	opts.BuildTags = "\n//go:build linux || darwin || synthetics\n"
	return devtools.GenerateIncludeListGo(opts)
}

// Update updates the generated files (aka make update).
func Update() {
	mg.SerialDeps(Fields, common.FieldDocs, Config, GenerateModuleIncludeListGo)
}

func IntegTest() {
	mg.SerialDeps(GoIntegTest)
}

func PythonIntegTest() {
	// intentionally blank, CI runs this for every beat
}

func GoIntegTest(ctx context.Context) error {
	return devtools.GoIntegTestFromHost(ctx, devtools.DefaultGoTestIntegrationFromHostArgs(ctx))
}

func Fields() error {
	return heartbeat.Fields()
}

// Config generates both the short/reference/docker configs.
func Config() error {
	return devtools.Config(devtools.AllConfigTypes, heartbeat.ConfigFileParams(), ".")
}

// syntheticsPinnedCommit is the @elastic/synthetics commit that introduces the
// apiJourney(...) DSL (elastic/synthetics#997). apiJourney is unreleased, so the
// api monitor scenario tests need this specific build. Once it ships in a
// release, drop InstallSyntheticsAgent and install @elastic/synthetics@<release>.
const syntheticsPinnedCommit = "f2bf619857d44694267983deeb830da551ad3d9e"

// InstallSyntheticsAgent installs the apiJourney-capable @elastic/synthetics
// agent globally so the api monitor scenario tests (ELASTIC_SYNTHETICS_API_CAPABLE)
// can run the same way in CI and locally. npm's git-dependency install of the
// pinned commit fails with ENOTDIR during its prepare step, so we clone, build,
// and install from the local path instead. Override the commit with the
// SYNTHETICS_COMMIT env var.
func InstallSyntheticsAgent() error {
	commit := syntheticsPinnedCommit
	if c := os.Getenv("SYNTHETICS_COMMIT"); c != "" {
		commit = c
	}

	// Clone into a stable, persistent path (do NOT delete it after installing):
	// `npm install -g .` links the global agent back to this checkout's built dist,
	// so removing it leaves the agent running unbuilt src/*.ts and inline api
	// journeys fail with "Cannot use import statement outside a module".
	dir := filepath.Join(os.TempDir(), "synthetics-pinned")
	if err := os.RemoveAll(dir); err != nil {
		return err
	}

	steps := []struct {
		dir  string
		name string
		args []string
	}{
		{"", "git", []string{"clone", "--filter=blob:none", "https://github.com/elastic/synthetics.git", dir}},
		{dir, "git", []string{"checkout", commit}},
		{dir, "npm", []string{"install"}},
		{dir, "npm", []string{"run", "build"}},
		{dir, "npm", []string{"install", "-g", "."}},
		{"", "elastic-synthetics", []string{"--version"}},
	}
	for _, s := range steps {
		cmd := exec.Command(s.name, s.args...)
		cmd.Dir = s.dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("installing synthetics agent (%s %s): %w", s.name, strings.Join(s.args, " "), err)
		}
	}
	return nil
}
