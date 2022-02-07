// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build mage
// +build mage

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"

	// mage:import
	"github.com/elastic/beats/v7/dev-tools/mage/target/common"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/docs"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/integtest/notests"
	// mage:import
	"github.com/elastic/beats/v7/dev-tools/mage/target/test"
)

const (
	goLintRepo     = "golang.org/x/lint/golint"
	goLicenserRepo = "github.com/elastic/go-licenser"
	buildDir       = "build"
	metaDir        = "_meta"
	snapshotEnv    = "SNAPSHOT"
	devEnv         = "DEV"
	configFile     = "elastic-agent.yml"
	agentDropPath  = "AGENT_DROP_PATH"
)

// Aliases for commands required by master makefile
var Aliases = map[string]interface{}{
	"build": Build.All,
	"demo":  Demo.Enroll,
}

func init() {
	common.RegisterCheckDeps(Update, Check.All)
	test.RegisterDeps(UnitTest)

	devtools.BeatDescription = "Agent manages other beats based on configuration provided."
	devtools.BeatLicense = "Elastic License"

	devtools.Platforms = devtools.Platforms.Filter("!linux/386")
	devtools.Platforms = devtools.Platforms.Filter("!windows/386")
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

// Demo runs agent out of container.
type Demo mg.Namespace

// Dev runs package and build for dev purposes.
type Dev mg.Namespace

// Env returns information about the environment.
func (Prepare) Env() {
	mg.Deps(Mkdir("build"), Build.GenerateConfig)
	RunGo("version")
	RunGo("env")
}

// Build builds the agent binary with DEV flag set.
func (Dev) Build() {
	dev := os.Getenv(devEnv)
	defer os.Setenv(devEnv, dev)

	os.Setenv(devEnv, "true")
	devtools.DevBuild = true
	mg.Deps(Build.All)
}

// Package packages the agent binary with DEV flag set.
func (Dev) Package() {
	dev := os.Getenv(devEnv)
	defer os.Setenv(devEnv, dev)

	os.Setenv(devEnv, "true")
	devtools.DevBuild = true
	Package()
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

// GenerateConfig generates the configuration from _meta/elastic-agent.yml
func (Build) GenerateConfig() error {
	mg.Deps(Mkdir(buildDir))
	return sh.Copy(filepath.Join(buildDir, configFile), filepath.Join(metaDir, configFile))
}

// GolangCrossBuildOSS build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuildOSS() error {
	params := devtools.DefaultGolangCrossBuildArgs()
	injectBuildVars(params.Vars)
	return devtools.GolangCrossBuild(params)
}

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	params := devtools.DefaultGolangCrossBuildArgs()
	params.OutputDir = "build/golang-crossbuild"
	injectBuildVars(params.Vars)

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
	buildArgs := devtools.DefaultBuildArgs()
	buildArgs.Name = "elastic-agent-oss"
	buildArgs.OutputDir = buildDir
	injectBuildVars(buildArgs.Vars)

	return devtools.Build(buildArgs)
}

// Binary build the fleet artifact.
func (Build) Binary() error {
	mg.Deps(Prepare.Env)

	buildArgs := devtools.DefaultBuildArgs()
	buildArgs.OutputDir = buildDir
	injectBuildVars(buildArgs.Vars)

	return devtools.Build(buildArgs)
}

// Clean up dev environment.
func (Build) Clean() {
	os.RemoveAll(buildDir)
}

// TestBinaries build the required binaries for the test suite.
func (Build) TestBinaries() error {
	p := filepath.Join("pkg", "agent", "operation", "tests", "scripts")
	p2 := filepath.Join("pkg", "agent", "transpiler", "tests")
	configurableName := "configurable"
	serviceableName := "serviceable"
	execName := "exec"
	if runtime.GOOS == "windows" {
		configurableName += ".exe"
		serviceableName += ".exe"
		execName += ".exe"
	}
	return combineErr(
		RunGo("build", "-o", filepath.Join(p, "configurable-1.0-darwin-x86_64", configurableName), filepath.Join(p, "configurable-1.0-darwin-x86_64", "main.go")),
		RunGo("build", "-o", filepath.Join(p, "serviceable-1.0-darwin-x86_64", serviceableName), filepath.Join(p, "serviceable-1.0-darwin-x86_64", "main.go")),
		RunGo("build", "-o", filepath.Join(p2, "exec-1.0-darwin-x86_64", execName), filepath.Join(p2, "exec-1.0-darwin-x86_64", "main.go")),
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

// Changes run git status --porcelain and return an error if we have changes or uncommitted files.
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
func (Test) Unit(ctx context.Context) error {
	mg.Deps(Prepare.Env, Build.TestBinaries)
	params := devtools.DefaultGoTestUnitArgs()
	return devtools.GoTest(ctx, params)
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

	platformPackages := []struct {
		platform string
		packages string
	}{
		{"darwin/amd64", "darwin-x86_64.tar.gz"},
		{"linux/amd64", "linux-x86_64.tar.gz"},
		{"linux/arm64", "linux-aarch64.tar.gz"},
		{"windows/amd64", "windows-x86_64.zip"},
	}

	var requiredPackages []string
	for _, p := range platformPackages {
		if _, enabled := devtools.Platforms.Get(p.platform); enabled {
			requiredPackages = append(requiredPackages, p.packages)
		}
	}

	if len(requiredPackages) == 0 {
		panic("elastic-agent package is expected to include other packages")
	}

	packageAgent(requiredPackages, devtools.UseElasticAgentPackaging)
}

func requiredPackagesPresent(basePath, beat, version string, requiredPackages []string) bool {
	for _, pkg := range requiredPackages {
		if _, ok := os.LookupEnv(snapshotEnv); ok {
			version += "-SNAPSHOT"
		}
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

// Update is an alias for executing control protocol, configs, and specs.
func Update() {
	mg.SerialDeps(Config, BuildSpec, BuildPGP, BuildFleetCfg)
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

// ControlProto generates pkg/agent/control/proto module.
func ControlProto() error {
	return sh.RunV("protoc", "--go_out=plugins=grpc:.", "control.proto")
}

// BuildSpec make sure that all the suppported program spec are built into the binary.
func BuildSpec() error {
	// go run x-pack/elastic-agent/dev-tools/cmd/buildspec/buildspec.go --in x-pack/agent/spec/*.yml --out x-pack/elastic-agent/pkg/agent/program/supported.go
	goF := filepath.Join("dev-tools", "cmd", "buildspec", "buildspec.go")
	in := filepath.Join("spec", "*.yml")
	out := filepath.Join("pkg", "agent", "program", "supported.go")

	fmt.Printf(">> Buildspec from %s to %s\n", in, out)
	return RunGo("run", goF, "--in", in, "--out", out)
}

func BuildPGP() error {
	// go run x-pack/elastic-agent/dev-tools/cmd/buildpgp/build_pgp.go --in x-pack/agent/spec/GPG-KEY-elasticsearch --out x-pack/elastic-agent/pkg/release/pgp.go
	goF := filepath.Join("dev-tools", "cmd", "buildpgp", "build_pgp.go")
	in := "GPG-KEY-elasticsearch"
	out := filepath.Join("pkg", "release", "pgp.go")

	fmt.Printf(">> BuildPGP from %s to %s\n", in, out)
	return RunGo("run", goF, "--in", in, "--out", out)
}

func configYML() error {
	return devtools.Config(devtools.AllConfigTypes, ConfigFileParams(), ".")
}

// ConfigFileParams returns the parameters for generating OSS config.
func ConfigFileParams() devtools.ConfigFileParams {
	p := devtools.DefaultConfigFileParams()
	p.Templates = append(p.Templates, "_meta/config/*.tmpl")
	p.Short.Template = "_meta/config/elastic-agent.yml.tmpl"
	p.Reference.Template = "_meta/config/elastic-agent.reference.yml.tmpl"
	p.Docker.Template = "_meta/config/elastic-agent.docker.yml.tmpl"
	return p
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

// UnitTest performs unit test on agent.
func UnitTest() {
	mg.Deps(Test.All)
}

// BuildFleetCfg embed the default fleet configuration as part of the binary.
func BuildFleetCfg() error {
	goF := filepath.Join("dev-tools", "cmd", "buildfleetcfg", "buildfleetcfg.go")
	in := filepath.Join("_meta", "elastic-agent.fleet.yml")
	out := filepath.Join("pkg", "agent", "application", "configuration_embed.go")

	fmt.Printf(">> BuildFleetCfg %s to %s\n", in, out)
	return RunGo("run", goF, "--in", in, "--out", out)
}

// Enroll runs agent which enrolls before running.
func (Demo) Enroll() error {
	env := map[string]string{
		"FLEET_ENROLL": "1",
	}
	return runAgent(env)
}

// Enroll runs agent which does not enroll before running.
func (Demo) NoEnroll() error {
	env := map[string]string{
		"FLEET_ENROLL": "0",
	}
	return runAgent(env)
}

func runAgent(env map[string]string) error {
	prevPlatforms := os.Getenv("PLATFORMS")
	defer os.Setenv("PLATFORMS", prevPlatforms)

	// setting this improves build time
	os.Setenv("PLATFORMS", "+all linux/amd64")
	devtools.Platforms = devtools.NewPlatformList("+all linux/amd64")

	supportedEnvs := map[string]int{"FLEET_ENROLLMENT_TOKEN": 0, "FLEET_ENROLL": 0, "FLEET_SETUP": 0, "FLEET_TOKEN_NAME": 0, "KIBANA_HOST": 0, "KIBANA_PASSWORD": 0, "KIBANA_USERNAME": 0}

	tag := dockerTag()
	dockerImageOut, err := sh.Output("docker", "image", "ls")
	if err != nil {
		return err
	}

	// docker does not exist for this commit, build it
	if !strings.Contains(dockerImageOut, tag) {
		// produce docker package
		packageAgent([]string{
			"linux-x86_64.tar.gz",
		}, devtools.UseElasticAgentDemoPackaging)

		dockerPackagePath := filepath.Join("build", "package", "elastic-agent", "elastic-agent-linux-amd64.docker", "docker-build")
		if err := os.Chdir(dockerPackagePath); err != nil {
			return err
		}

		// build docker image
		if err := dockerBuild(tag); err != nil {
			fmt.Println(">> Building docker images again (after 10 seconds)")
			// This sleep is to avoid hitting the docker build issues when resources are not available.
			time.Sleep(10)
			if err := dockerBuild(tag); err != nil {
				return err
			}
		}
	}

	// prepare env variables
	envs := []string{
		// providing default kibana to be fixed for os-es if not provided
		"KIBANA_HOST=http://localhost:5601",
	}

	envs = append(envs, os.Environ()...)
	for k, v := range env {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	// run docker cmd
	dockerCmdArgs := []string{"run", "--network", "host"}
	for _, e := range envs {
		parts := strings.SplitN(e, "=", 2)
		if _, isSupported := supportedEnvs[parts[0]]; !isSupported {
			continue
		}

		// fix value
		e = fmt.Sprintf("%s=%s", parts[0], fixOsEnv(parts[0], parts[1]))

		dockerCmdArgs = append(dockerCmdArgs, "-e", e)
	}

	dockerCmdArgs = append(dockerCmdArgs, tag)
	return sh.Run("docker", dockerCmdArgs...)
}

func packageAgent(requiredPackages []string, packagingFn func()) {
	version, found := os.LookupEnv("BEAT_VERSION")
	if !found {
		version = release.Version()
	}

	// build deps only when drop is not provided
	if dropPathEnv, found := os.LookupEnv(agentDropPath); !found || len(dropPathEnv) == 0 {
		// prepare new drop
		dropPath := filepath.Join("build", "distributions", "elastic-agent-drop")
		dropPath, err := filepath.Abs(dropPath)
		if err != nil {
			panic(err)
		}

		if err := os.MkdirAll(dropPath, 0755); err != nil {
			panic(err)
		}

		os.Setenv(agentDropPath, dropPath)
		if runtime.GOARCH == "arm64" {
			const platformsVar = "PLATFORMS"
			oldPlatforms := os.Getenv(platformsVar)
			os.Setenv(platformsVar, runtime.GOOS+"/"+runtime.GOARCH)
			if oldPlatforms != "" {
				defer os.Setenv(platformsVar, oldPlatforms)
			} else {
				defer os.Unsetenv(oldPlatforms)
			}
		}

		// cleanup after build
		defer os.RemoveAll(dropPath)
		defer os.Unsetenv(agentDropPath)

		packedBeats := []string{"filebeat", "heartbeat", "cloudbeat", "metricbeat"}

		for _, b := range packedBeats {
			pwd, err := filepath.Abs(filepath.Join("..", b))
			if err != nil {
				panic(err)
			}

			if !requiredPackagesPresent(pwd, b, version, requiredPackages) {
				cmd := exec.Command("mage", "package")
				cmd.Dir = pwd
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Env = append(os.Environ(), fmt.Sprintf("PWD=%s", pwd), "AGENT_PACKAGING=on")
				if envVar := selectedPackageTypes(); envVar != "" {
					cmd.Env = append(cmd.Env, envVar)
				}

				if err := cmd.Run(); err != nil {
					panic(err)
				}
			}

			// copy to new drop
			sourcePath := filepath.Join(pwd, "build", "distributions")
			if err := copyAll(sourcePath, dropPath); err != nil {
				panic(err)
			}
		}
	}

	// package agent
	packagingFn()

	mg.Deps(Update)
	mg.Deps(CrossBuild, CrossBuildGoDaemon)
	mg.SerialDeps(devtools.Package, TestPackages)
}

func selectedPackageTypes() string {
	if len(devtools.SelectedPackageTypes) == 0 {
		return ""
	}

	return "PACKAGES=targz,zip"
}

func copyAll(from, to string) error {
	return filepath.Walk(from, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		targetFile := filepath.Join(to, info.Name())

		// overwrites with current build
		return sh.Copy(targetFile, path)
	})
}

func dockerBuild(tag string) error {
	return sh.Run("docker", "build", "-t", tag, ".")
}

func dockerTag() string {
	const commitLen = 7
	tagBase := "elastic-agent"

	commit, err := devtools.CommitHash()
	if err == nil && len(commit) > commitLen {
		return fmt.Sprintf("%s-%s", tagBase, commit[:commitLen])
	}

	return tagBase
}

func fixOsEnv(k, v string) string {
	switch k {
	case "KIBANA_HOST":
		// network host works in a weird way here
		if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
			return strings.Replace(strings.ToLower(v), "localhost", "host.docker.internal", 1)
		}
	}

	return v
}

func buildVars() map[string]string {
	vars := make(map[string]string)

	isSnapshot, _ := os.LookupEnv(snapshotEnv)
	vars["github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release.snapshot"] = isSnapshot

	if isDevFlag, devFound := os.LookupEnv(devEnv); devFound {
		if isDev, err := strconv.ParseBool(isDevFlag); err == nil && isDev {
			vars["github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release.allowEmptyPgp"] = "true"
			vars["github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release.allowUpgrade"] = "true"
		}
	}

	return vars
}

func injectBuildVars(m map[string]string) {
	for k, v := range buildVars() {
		m[k] = v
	}
}
