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

// +build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/elastic/fleet/dev-tools/mage"
	"github.com/hashicorp/go-multierror"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	goLintRepo     = "golang.org/x/lint/golint"
	goLicenserRepo = "github.com/elastic/go-licenser"
	buildDir       = "build"
	metaDir        = "_meta"
)

// Default set to build everything by default.
var Default = Build.All

// Build namespace used to build binaries.
type Build mg.Namespace

// Test namespace contains all the task for testing the projects.
type Test mg.Namespace

// Check namespace contains tasks related check the actual code quality.
type Check mg.Namespace

// Prepare tasks related to bootstrap the environment or get information about the environment.
type Prepare mg.Namespace

// Format automatically format the code.
type Format mg.Namespace

// Env returns information about the environment.
func (Prepare) Env() {
	mg.Deps(Mkdir("build"), Build.GenerateConfig)
	RunGo("version")
	RunGo("env")
}

// InstallGoLicenser install go-licenser to check license of the files.
func (Prepare) InstallGoLicenser() error {
	return GoGet(goLicenserRepo)
}

// InstallGoLint for the code.
func (Prepare) InstallGoLint() error {
	return GoGet(goLintRepo)
}

// All build all the things for the current projects.
func (Build) All() {
	mg.Deps(Build.Binary)
}

// GenerateConfig generates the configuration from _meta/agent.yml
func (Build) GenerateConfig() error {
	mg.Deps(Mkdir(buildDir))
	return sh.Copy(filepath.Join(buildDir, "agent.yml"), filepath.Join(metaDir, "agent.yml"))
}

// GolangCrossBuildOSS build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuildOSS() error {
	params := mage.DefaultGolangCrossBuildArgs()
	params.InputFile = "cmd/agent/agent.go"
	return mage.GolangCrossBuild(params)
}

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	params := mage.DefaultGolangCrossBuildArgs()
	params.InputFile = "x-pack/cmd/agent/agent.go"
	params.OutputDir = "x-pack/build/golang-crossbuild"
	if err := mage.GolangCrossBuild(params); err != nil {
		return err
	}

	// TODO: no OSS bits just yet
	// return GolangCrossBuildOSS()

	return nil
}

// BuildGoDaemon builds the go-daemon binary (use crossBuildGoDaemon).
func BuildGoDaemon() error {
	return mage.BuildGoDaemon()
}

// Binary build the fleet artifact.
func (Build) BinaryOSS() error {
	mg.Deps(Prepare.Env)
	return RunGo(
		"build",
		"-o", filepath.Join(buildDir, "agent-oss"),
		"-ldflags", flags(),
		"-i", "cmd/agent/agent.go",
	)
}

// Binary build the fleet artifact.
func (Build) Binary() error {
	mg.Deps(Prepare.Env)
	return RunGo(
		"build",
		"-o", filepath.Join(buildDir, "agent"),
		"-ldflags", flags(),
		"-i", "x-pack/cmd/agent/agent.go",
	)
}

// Clean up dev environment.
func (Build) Clean() {
	os.RemoveAll(buildDir)
}

// All run all the code checks.
func (Check) All() {
	mg.SerialDeps(Check.License, Check.GoLint)
}

// GoLint run the code through the linter.
func (Check) GoLint() error {
	mg.Deps(Prepare.InstallGoLint)
	packagesString, err := sh.Output("go", "list", "./...")
	if err != nil {
		return err
	}

	packages := strings.Split(packagesString, "\n")
	for _, pkg := range packages {
		if strings.Contains(pkg, "/vendor/") {
			continue
		}

		if e := sh.RunV("golint", "-set_exit_status", pkg); e != nil {
			err = multierror.Append(err, e)
		}
	}

	return err
}

// License makes sure that all the Golang files have the appropriate license header.
func (Check) License() error {
	mg.Deps(Prepare.InstallGoLicenser)
	// exclude copied files until we come up with a better option
	return sh.RunV("go-licenser", "-d", "-license", "ASL2", "-exclude", "x-pack")
	return sh.RunV("go-licenser", "-d", "-license", "Elastic", "x-pack")
}

// All runs all the tests.
func (Test) All() {
	mg.SerialDeps(Test.Unit)
}

// Unit runs all the unit tests.
func (Test) Unit() error {
	mg.Deps(Prepare.Env)
	return RunGo("test", "-race", "-v", "-coverprofile", filepath.Join(buildDir, "coverage.out"), "./...")
}

// Coverage takes the coverages report from running all the tests and display the results in the browser.
func (Test) Coverage() error {
	mg.Deps(Prepare.Env)
	return RunGo("tool", "cover", "-html="+filepath.Join(buildDir, "coverage.out"))
}

// All format automatically all the codes.
func (Format) All() {
	mg.SerialDeps(Format.License)
}

// License applies the right license header.
func (Format) License() error {
	mg.Deps(Prepare.InstallGoLicenser)
	return sh.RunV("go-licenser", "-license", "ASL2", "-exclude", "x-pack")
	return sh.RunV("go-licenser", "-license", "Elastic", "x-pack")
}

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
// Use VERSION_QUALIFIER to control the version qualifier.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()
	//mage.UseElasticBeatOSSPackaging()
	mage.UseElasticBeatPackaging()

	mg.Deps(Update)
	mg.Deps(CrossBuild, CrossBuildGoDaemon)
	mg.SerialDeps(mage.Package, TestPackages)
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return mage.TestPackages()
}

// RunGo runs go command and output the feedback to the stdout and the stderr.
func RunGo(args ...string) error {
	return sh.RunV(mg.GoCmd(), args...)
}

// GoGet fetch a remote dependencies.
func GoGet(link string) error {
	_, err := sh.Exec(map[string]string{"GO111MODULE": "off"}, os.Stdout, os.Stderr, "go", "get", link)
	return err
}

// Mkdir returns a function that create a directory.
func Mkdir(dir string) func() error {
	return func() error {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create directory: %v, error: %+v", dir, err)
		}
		return nil
	}
}

func commitID() string {
	commitID, err := sh.Output("git", "rev-parse", "--short", "HEAD")
	if err != nil {
		return "cannot retrieve hash"
	}
	return commitID
}

func flags() string {
	ts := time.Now().Format(time.RFC3339)
	commitID := commitID()

	return fmt.Sprintf(
		`-X "github.com/elastic/fleet/pkg/release.buildTime=%s" -X "github.com/elastic/fleet/pkg/release.commit=%s"`,
		ts,
		commitID,
	)
}

// Update is an alias for executing fields, dashboards, config, includes.
func Update() {
	mg.SerialDeps(Config) //, fieldDocs)
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	return mage.CrossBuild()
}

// CrossBuildGoDaemon cross-builds the go-daemon binary using Docker.
func CrossBuildGoDaemon() error {
	return mage.CrossBuildGoDaemon()
}

// Config generates both the short/reference/docker configs and populates the
// modules.d directory.
func Config() {
	mg.Deps(configYML)
}

func configYML() error {
	return mage.Config(mage.AllConfigTypes, OSSConfigFileParams(), ".")
}

// OSSConfigFileParams returns the parameters for generating OSS config.
func OSSConfigFileParams() mage.ConfigFileParams {
	return mage.ConfigFileParams{
		ShortParts: []string{
			mage.OSSBeatDir("_meta/common.p1.yml"),
			mage.OSSBeatDir("_meta/common.p2.yml"),
		},
		ReferenceParts: []string{
			mage.OSSBeatDir("_meta/common.reference.p1.yml"),
			mage.OSSBeatDir("_meta/common.reference.p2.yml"),
		},
		DockerParts: []string{
			mage.OSSBeatDir("_meta/agent.docker.yml"),
		},
	}
}

// fieldDocs generates docs/fields.asciidoc containing all fields
// (including x-pack).
func fieldDocs() error {
	inputs := []string{
		mage.OSSBeatDir("input"),
		mage.XPackBeatDir("input"),
	}
	output := mage.CreateDir("build/fields/fields.all.yml")
	if err := mage.GenerateFieldsYAMLTo(output, inputs...); err != nil {
		return err
	}
	return mage.Docs.FieldDocs(output)
}
