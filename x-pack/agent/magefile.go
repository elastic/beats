// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build mage

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	devtools "github.com/elastic/beats/dev-tools/mage"
	"github.com/elastic/beats/dev-tools/mage/target/common"
	"github.com/elastic/beats/dev-tools/mage/target/unittest"
	"github.com/elastic/beats/x-pack/agent/pkg/release"

	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/common"

	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/docs"
)

const (
	goLintRepo     = "golang.org/x/lint/golint"
	goLicenserRepo = "github.com/elastic/go-licenser"
	buildDir       = "build"
	metaDir        = "_meta"
)

// Aliases for commands required by master makefile
var Aliases = map[string]interface{}{
	"build": Build.All,
}

func init() {
	common.RegisterCheckDeps(Update, Check.All)

	devtools.BeatDescription = "Agent manages other beats based on configuration provided."
	devtools.BeatLicense = "Elastic License"
}

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
	params := devtools.DefaultGolangCrossBuildArgs()
	params.InputFiles = []string{"cmd/agent/agent.go"}
	return devtools.GolangCrossBuild(params)
}

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	params := devtools.DefaultGolangCrossBuildArgs()
	params.InputFiles = []string{"cmd/agent/agent.go"}
	params.OutputDir = "build/golang-crossbuild"
	if err := devtools.GolangCrossBuild(params); err != nil {
		return err
	}

	// TODO: no OSS bits just yet
	// return GolangCrossBuildOSS()

	return nil
}

// BuildGoDaemon builds the go-daemon binary (use crossBuildGoDaemon).
func BuildGoDaemon() error {
	return devtools.BuildGoDaemon()
}

// BinaryOSS build the fleet artifact.
func (Build) BinaryOSS() error {
	mg.Deps(Prepare.Env)
	return RunGo(
		"build",
		"-o", filepath.Join(buildDir, "agent-oss"),
		"-ldflags", flags(),
		"cmd/agent/agent.go",
	)
}

// Binary build the fleet artifact.
func (Build) Binary() error {
	mg.Deps(Prepare.Env)
	return RunGo(
		"build",
		"-o", filepath.Join(buildDir, "agent"),
		"-ldflags", flags(),
		"cmd/agent/agent.go",
	)
}

// Dev make a special build with the Dev tags.
func (Build) Dev() error {
	mg.Deps(Prepare.Env)
	return RunGo(
		"build",
		"-tags", "dev",
		"-o", filepath.Join(buildDir, "agent"),
		"-ldflags", flags(),
		"cmd/agent/agent.go",
	)
}

// Clean up dev environment.
func (Build) Clean() {
	os.RemoveAll(buildDir)
}

// TestBinaries build the required binaries for the test suite.
func (Build) TestBinaries() error {
	p := filepath.Join("pkg", "agent", "operation", "tests", "scripts")

	return combineErr(
		RunGo("build", "-o", filepath.Join(p, "configurable-1.0-darwin-x86", "configurable"), filepath.Join(p, "configurable-1.0-darwin-x86", "main.go")),
		RunGo("build", "-o", filepath.Join(p, "configurablebyfile-1.0-darwin-x86", "configurablebyfile"), filepath.Join(p, "configurablebyfile-1.0-darwin-x86", "main.go")),
	)
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
	return combineErr(
		sh.RunV("go-licenser", "-d", "-license", "Elastic"),
	)
}

// Changes run git status --porcelain and return an error if we have changes or uncommited files.
func (Check) Changes() error {
	out, err := sh.Output("git", "status", "--porcelain")
	if err != nil {
		return errors.New("cannot retrieve hash")
	}

	if len(out) != 0 {
		fmt.Fprintln(os.Stderr, "Changes:")
		fmt.Fprintln(os.Stderr, out)
		return fmt.Errorf("uncommited changes")
	}
	return nil
}

// All runs all the tests.
func (Test) All() {
	mg.SerialDeps(Test.Unit)
}

// Unit runs all the unit tests.
func (Test) Unit() error {
	mg.Deps(Prepare.Env, Build.TestBinaries)
	return RunGo("test", "-race", "-v", "-coverprofile", filepath.Join(buildDir, "coverage.out"), "./...")
}

// Coverage takes the coverages report from running all the tests and display the results in the browser.
func (Test) Coverage() error {
	mg.Deps(Prepare.Env, Build.TestBinaries)
	return RunGo("tool", "cover", "-html="+filepath.Join(buildDir, "coverage.out"))
}

// All format automatically all the codes.
func (Format) All() {
	mg.SerialDeps(Format.License)
}

// License applies the right license header.
func (Format) License() error {
	mg.Deps(Prepare.InstallGoLicenser)
	return combineErr(
		sh.RunV("go-licenser", "-license", "Elastic"),
	)
}

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
// Use VERSION_QUALIFIER to control the version qualifier.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	version, found := os.LookupEnv("BEAT_VERSION")
	if !found {
		version = release.Version()
	}

	packedBeats := []string{"filebeat", "metricbeat"}
	requiredPackages := []string{
		"darwin-x86_64.tar.gz",
		"linux-x86.tar.gz",
		"linux-x86_64.tar.gz",
		"windows-x86.zip",
		"windows-x86_64.zip",
	}

	for _, b := range packedBeats {
		pwd, err := filepath.Abs(filepath.Join("..", b))
		if err != nil {
			panic(err)
		}

		if requiredPackagesPresent(pwd, b, version, requiredPackages) {
			continue
		}

		cmd := exec.Command("mage", "package")
		cmd.Dir = pwd
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = append(os.Environ(), fmt.Sprintf("PWD=%s", pwd), "AGENT_PACKAGING=on")

		if err := cmd.Run(); err != nil {
			panic(err)
		}
	}

	// package agent
	devtools.UseElasticBeatAgentPackaging()

	mg.Deps(Update)
	mg.Deps(CrossBuild, CrossBuildGoDaemon)
	mg.SerialDeps(devtools.Package)
}

func requiredPackagesPresent(basePath, beat, version string, requiredPackages []string) bool {
	for _, pkg := range requiredPackages {
		packageName := fmt.Sprintf("%s-%s-%s", beat, version, pkg)
		path := filepath.Join(basePath, "build", "distributions", packageName)

		if _, err := os.Stat(path); err != nil {
			fmt.Printf("Package '%s' does not exist on path: %s\n", packageName, path)
			return false
		}
	}
	return true
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return devtools.TestPackages()
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
		`-X "github.com/elastic/beats/x-pack/agent/pkg/release.buildTime=%s" -X "github.com/elastic/beats/x-pack/agent/pkg/release.commit=%s"`,
		ts,
		commitID,
	)
}

// Update is an alias for executing fields, dashboards, config, includes.
func Update() {
	mg.SerialDeps(Config, BuildSpec, BuildFleetCfg)
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	return devtools.CrossBuild()
}

// CrossBuildGoDaemon cross-builds the go-daemon binary using Docker.
func CrossBuildGoDaemon() error {
	return devtools.CrossBuildGoDaemon()
}

// Config generates both the short/reference/docker.
func Config() {
	mg.Deps(configYML)
}

// BuildSpec make sure that all the suppported program spec are built into the binary.
func BuildSpec() error {
	// go run x-pack/agent/dev-tools/cmd/buildspec/buildspec.go --in x-pack/agent/spec/*.yml --out x-pack/agent/pkg/agent/program/supported.go
	goF := filepath.Join("dev-tools", "cmd", "buildspec", "buildspec.go")
	in := filepath.Join("spec", "*.yml")
	out := filepath.Join("pkg", "agent", "program", "supported.go")

	fmt.Printf(">> Buildspec from %s to %s\n", in, out)
	return RunGo("run", goF, "--in", in, "--out", out)
}

func configYML() error {
	return devtools.Config(devtools.AllConfigTypes, ConfigFileParams(), ".")
}

// ConfigFileParams returns the parameters for generating OSS config.
func ConfigFileParams() devtools.ConfigFileParams {
	return devtools.ConfigFileParams{
		ShortParts: []string{
			devtools.XPackBeatDir("_meta/common.p1.yml"),
			devtools.XPackBeatDir("_meta/common.p2.yml"),
		},
		ReferenceParts: []string{
			devtools.XPackBeatDir("_meta/common.reference.p1.yml"),
			devtools.XPackBeatDir("_meta/common.reference.p2.yml"),
		},
		DockerParts: []string{
			devtools.XPackBeatDir("_meta/agent.docker.yml"),
		},
	}
}

// fieldDocs generates docs/fields.asciidoc containing all fields
// (including x-pack).
func fieldDocs() error {
	inputs := []string{
		devtools.OSSBeatDir("input"),
		devtools.XPackBeatDir("input"),
	}
	output := devtools.CreateDir("build/fields/fields.all.yml")
	if err := devtools.GenerateFieldsYAMLTo(output, inputs...); err != nil {
		return err
	}
	return devtools.Docs.FieldDocs(output)
}

func combineErr(errors ...error) error {
	var e error
	for _, err := range errors {
		if err == nil {
			continue
		}
		e = multierror.Append(e, err)
	}
	return e
}

// GoTestUnit is an alias for goUnitTest.
func GoTestUnit() {
	mg.Deps(unittest.GoUnitTest)
}

// UnitTest performs unit test on agent.
func UnitTest() {
	mg.Deps(Test.All)
}

// IntegTest calls go integtest, we dont have python integ test so far
// TODO: call integtest mage package when python tests are available
func IntegTest() {
	os.Create(filepath.Join("build", "TEST-go-integration.out"))
}

// BuildFleetCfg embed the default fleet configuration as part of the binary.
func BuildFleetCfg() error {
	goF := filepath.Join("dev-tools", "cmd", "buildfleetcfg", "buildfleetcfg.go")
	in := filepath.Join("_meta", "agent.fleet.yml")
	out := filepath.Join("pkg", "agent", "application", "configuration_embed.go")

	fmt.Printf(">> BuildFleetCfg %s to %s\n", in, out)
	return RunGo("run", goF, "--in", in, "--out", out)
}
