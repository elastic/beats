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

package mage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"golang.org/x/sys/execabs"

	"github.com/elastic/beats/v7/dev-tools/mage/gotool"
)

// GoTestArgs are the arguments used for the "go*Test" targets and they define
// how "go test" is invoked. "go test" is always invoked with -v for verbose.
type GoTestArgs struct {
	TestName            string            // Test name used in logging.
	Race                bool              // Enable race detector.
	Tags                []string          // Build tags to enable.
	ExtraFlags          []string          // Extra flags to pass to 'go test'.
	Packages            []string          // Packages to test.
	Env                 map[string]string // Env vars to add to the current env.
	OutputFile          string            // File to write verbose test output to.
	JUnitReportFile     string            // File to write a JUnit XML test report to.
	CoverageProfileFile string            // Test coverage profile file (enables -cover).
	Output              io.Writer         // Write stderr and stdout to Output if set
}

// TestBinaryArgs are the arguments used when building binary for testing.
type TestBinaryArgs struct {
	Name       string // Name of the binary to build
	InputFiles []string
	ExtraFlags []string // Extra flags to pass to 'go test'.
}

func makeGoTestArgs(name string) GoTestArgs {
	fileName := fmt.Sprintf("build/TEST-go-%s", strings.ReplaceAll(strings.ToLower(name), " ", "_"))
	params := GoTestArgs{
		TestName:        name,
		Race:            RaceDetector,
		Packages:        []string{"./..."},
		Env:             make(map[string]string),
		OutputFile:      fileName + ".out",
		JUnitReportFile: fileName + ".xml",
		Tags:            testTagsFromEnv(),
	}
	if TestCoverage {
		params.CoverageProfileFile = fileName + ".cov"
	}
	return params
}

func makeGoTestArgsForPackage(name, pkg string) GoTestArgs {
	fileName := fmt.Sprintf(
		"build/TEST-go-%s-%s",
		strings.ReplaceAll(strings.ToLower(name), " ", "_"),
		strings.ReplaceAll(strings.ToLower(pkg), " ", "_"))
	params := GoTestArgs{
		TestName:        fmt.Sprintf("%s-%s", name, pkg),
		Race:            RaceDetector,
		Packages:        []string{fmt.Sprintf("./module/%s", pkg)},
		OutputFile:      fileName + ".out",
		JUnitReportFile: fileName + ".xml",
		Tags:            testTagsFromEnv(),
	}
	if TestCoverage {
		params.CoverageProfileFile = fileName + ".cov"
	}
	return params
}

// fetchGoPackages retrieves all Go packages for a beats module. It uses
// "go list -tags integration" to obtain the list of packages.
// Example: for the "kafka" module inside "metricbeat/module", it'll return:
//
//	[kafka kafka/broker kafka/consumer kafka/consumergroup kafka/partition kafka/producer]
func fetchGoPackages(module string) ([]string, error) {
	cmd := execabs.Command(
		"go", "list", "-tags", "integration", fmt.Sprintf("./%s/...", module))
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	rawPackages := strings.Split(strings.TrimSpace(string(output)), "\n")
	var pkgs []string
	for _, pkg := range rawPackages {
		tmp := strings.Split(pkg, "/module/")
		if len(tmp) != 2 {
			continue
		}

		pkgs = append(pkgs, tmp[1])
	}
	return pkgs, nil
}

// testTagsFromEnv gets a list of comma-separated tags from the TEST_TAGS
// environment variables, e.g: TEST_TAGS=aws,azure.
// If the FIPS env var is set to true, the requirefips tag is injected.
func testTagsFromEnv() []string {
	tags := strings.Split(strings.Trim(os.Getenv("TEST_TAGS"), ", "), ",")
	if FIPSBuild {
		tags = append(tags, "requirefips")
	}
	return tags
}

// DefaultGoTestUnitArgs returns a default set of arguments for running
// all unit tests. We tag unit test files with '!integration'.
func DefaultGoTestUnitArgs() GoTestArgs { return makeGoTestArgs("Unit") }

// DefaultGoFIPSOnlyTestArgs returns a default set of arguments for running
// fips140=only unit tests.
func DefaultGoFIPSOnlyTestArgs() GoTestArgs {
	args := makeGoTestArgs("Unit-FIPS-only")
	args.Env["GODEBUG"] = "fips140=only"
	return args
}

// DefaultGoTestIntegrationArgs returns a default set of arguments for running
// all integration tests. We tag integration test files with 'integration'.
func DefaultGoTestIntegrationArgs() GoTestArgs {
	args := makeGoTestArgs("Integration")
	args.Tags = append(args.Tags, "integration")

	synth := exec.Command("npx", "@elastic/synthetics", "-h")
	if synth.Run() == nil {
		// Run an empty journey to ensure playwright can be loaded
		// catches situations like missing playwright deps
		cmd := exec.Command("sh", "-c", "echo 'step(\"t\", () => { })' | elastic-synthetics --inline")
		var out strings.Builder
		cmd.Stdout = &out
		cmd.Stderr = &out
		err := cmd.Run()
		if err != nil || cmd.ProcessState.ExitCode() != 0 {
			fmt.Printf("synthetics is available, but not invokable, command exited with bad code: %s\n", out.String())
		}

		fmt.Println("npx @elastic/synthetics found, will run with synthetics tags")
		os.Setenv("ELASTIC_SYNTHETICS_CAPABLE", "true")
		args.Tags = append(args.Tags, "synthetics")
	}

	// Use the non-cachable -count=1 flag to disable test caching when running integration tests.
	// There are reasons to re-run tests even if the code is unchanged (e.g. Dockerfile changes).
	args.ExtraFlags = append(args.ExtraFlags, "-count=1")
	return args
}

// DefaultGoTestIntegrationFromHostArgs returns a default set of arguments for running
// all integration tests from the host system (outside the docker network).
func DefaultGoTestIntegrationFromHostArgs() GoTestArgs {
	args := DefaultGoTestIntegrationArgs()
	args.Env = WithGoIntegTestHostEnv(args.Env)
	return args
}

// GoTestIntegrationArgsForPackage returns a default set of arguments for running
// module integration tests. We tag integration test files with 'integration'.
func GoTestIntegrationArgsForPackage(pkg string) GoTestArgs {
	args := makeGoTestArgsForPackage("Integration", pkg)

	args.Tags = append(args.Tags, "integration")
	return args
}

// DefaultTestBinaryArgs returns the default arguments for building
// a binary for testing.
func DefaultTestBinaryArgs() TestBinaryArgs {
	return TestBinaryArgs{
		Name: BeatName,
	}
}

// GoTestIntegrationForModule executes the Go integration tests for each Go
// package within a module sequentially.
// Currently, all test cases must be present under "./module" directory.
//
// Motivation: previous implementation executed all integration tests at once,
// causing high CPU load, high memory usage and resulted in timeouts.
//
// This method executes integration tests for a single module at a time.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
// Use MODULE=module to run only tests for `module`.
func GoTestIntegrationForModule(ctx context.Context) error {
	modules := EnvOr("MODULE", "")
	if modules == "" {
		log.Printf("Warning: environment variable MODULE is empty: [%s]\n", modules)
	}
	moduleArr := strings.Split(modules, ",")

	for _, module := range moduleArr {
		err := goTestIntegrationForSingleModule(ctx, module)
		if err != nil {
			return err
		}
	}
	return nil
}

// goTestIntegrationForSingleModule sequentially executes the tests every Go
// packages within a module.
func goTestIntegrationForSingleModule(ctx context.Context, module string) error {
	modulesFileInfo, err := os.ReadDir("./module")
	if err != nil {
		return err
	}

	foundModule := false
	failedModules := make([]string, 0, len(modulesFileInfo))
	for _, fi := range modulesFileInfo {
		// skip the ones that are not directories or with suffix @tmp, which are created by Jenkins build job
		if !fi.IsDir() || strings.HasSuffix(fi.Name(), "@tmp") {
			continue
		}
		if module != "" && module != fi.Name() {
			continue
		}
		foundModule = true

		// Set MODULE because only want that modules tests to run inside the testing environment.
		env := map[string]string{"MODULE": fi.Name()}
		passThroughEnvs(env, IntegrationTestEnvVars()...)
		runners, err := NewIntegrationRunners(path.Join("./module", fi.Name()), env)
		if err != nil {
			return fmt.Errorf("test setup failed for module %s: %w", fi.Name(), err)
		}
		err = runners.Test("goIntegTest", func() error {
			pkgs, err := fetchGoPackages("module/" + fi.Name())
			if err != nil {
				return fmt.Errorf("could not list packages for module %s: %w",
					fi.Name(), err)
			}

			var errs []error
			for _, pkg := range pkgs {
				err := GoTest(ctx, GoTestIntegrationArgsForPackage(pkg))
				if err != nil {
					errs = append(errs, err)
				}
			}
			return errors.Join(errs...)
		})
		if err != nil {
			fmt.Printf("Error: failed to run integration tests for module %s:\n%v\n", fi.Name(), err)
			// err will already be report to stdout, collect failed module to report at end
			failedModules = append(failedModules, fi.Name())
		}
	}
	if module != "" && !foundModule {
		return fmt.Errorf("no module %s", module)
	}
	if len(failedModules) > 0 {
		return fmt.Errorf("failed modules: %s", strings.Join(failedModules, ", "))
	}
	return nil
}

// InstallGoTestTools installs additional tools that are required to run unit and integration tests.
func InstallGoTestTools() error {
	return gotool.Install(
		gotool.Install.Package("gotest.tools/gotestsum"),
	)
}

// GoTest invokes "go test" and reports the results to stdout. It returns an
// error if there was any failure executing the tests or if there were any
// test failures.
func GoTest(ctx context.Context, params GoTestArgs) error {
	mg.Deps(InstallGoTestTools)

	fmt.Println(">> go test:", params.TestName, "Testing")

	// We use gotestsum to drive the tests and produce a junit report.
	// The tool runs `go test -json` in order to produce a structured log which makes it easier
	// to parse the actual test output.
	// Of OutputFile is given the original JSON file will be written as well.
	//
	// The runner needs to set CLI flags for gotestsum and for "go test". We track the different
	// CLI flags in the gotestsumArgs and testArgs variables, such that we can finally produce command like:
	//   $ gotestsum <gotestsum args> -- <go test args>
	//
	// The additional arguments given via GoTestArgs are applied to `go test` only. Callers can not
	// modify any of the gotestsum arguments.

	gotestsumArgs := []string{"--no-color"}
	if mg.Verbose() {
		gotestsumArgs = append(gotestsumArgs, "-f", "standard-verbose")
	} else {
		gotestsumArgs = append(gotestsumArgs, "-f", "standard-quiet")
	}
	if params.JUnitReportFile != "" {
		CreateDir(params.JUnitReportFile)
		gotestsumArgs = append(gotestsumArgs, "--junitfile", params.JUnitReportFile)
	}
	if params.OutputFile != "" {
		CreateDir(params.OutputFile)
		gotestsumArgs = append(gotestsumArgs, "--jsonfile", params.OutputFile+".json")
	}

	var testArgs []string

	if params.Race {
		// Enable the race detector for supported platforms.
		// This is an intersection of the supported platforms for Beats and Go.
		//
		// See https://go.dev/doc/articles/race_detector#Requirements.
		devOS := os.Getenv("DEV_OS")
		devArch := os.Getenv("DEV_ARCH")
		raceAmd64 := devArch == "amd64"
		raceArm64 := devArch == "arm64" &&
			slices.Contains([]string{"linux", "darwin"}, devOS)
		if raceAmd64 || raceArm64 {
			testArgs = append(testArgs, "-race")
		} else {
			log.Printf("Warning: skipping -race flag for unsupported platform %s/%s\n", devOS, devArch)
		}
	}
	if len(params.Tags) > 0 {
		params := strings.Join(params.Tags, ",")
		if params != "" {
			testArgs = append(testArgs, "-tags="+params)
		}
	}
	if params.CoverageProfileFile != "" {
		params.CoverageProfileFile = createDir(filepath.Clean(params.CoverageProfileFile))
		testArgs = append(testArgs,
			"-covermode=atomic",
			"-coverprofile="+params.CoverageProfileFile,
		)
	}
	testArgs = append(testArgs, params.ExtraFlags...)
	testArgs = append(testArgs, params.Packages...)

	args := append(gotestsumArgs, append([]string{"--"}, testArgs...)...)

	goTest := makeCommand(ctx, params.Env, "gotestsum", args...)
	// Wire up the outputs.
	var outputs []io.Writer
	if params.Output != nil {
		outputs = append(outputs, params.Output)
	}

	if params.OutputFile != "" {
		fileOutput, err := os.Create(CreateDir(params.OutputFile))
		if err != nil {
			return fmt.Errorf("failed to create go test output file: %w", err)
		}
		defer fileOutput.Close()
		outputs = append(outputs, fileOutput)
	}
	output := io.MultiWriter(outputs...)
	if params.Output == nil {
		goTest.Stdout = io.MultiWriter(output, os.Stdout)
		goTest.Stderr = io.MultiWriter(output, os.Stderr)
	} else {
		goTest.Stdout = output
		goTest.Stderr = output
	}

	err := goTest.Run()

	var goTestErr *exec.ExitError
	if err != nil {
		// Command ran.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("failed to execute go: %w", err)
		}

		// Command ran but failed. Process the output.
		goTestErr = exitErr
	}

	if goTestErr != nil {
		// No packages were tested. Probably the code didn't compile.
		return fmt.Errorf("go test returned a non-zero value: %w", goTestErr)
	}

	// Generate a HTML code coverage report.
	var htmlCoverReport string
	if params.CoverageProfileFile != "" {

		htmlCoverReport = strings.TrimSuffix(params.CoverageProfileFile,
			filepath.Ext(params.CoverageProfileFile)) + ".html"

		coverToHTML := sh.RunCmd("go", "tool", "cover",
			"-html="+params.CoverageProfileFile,
			"-o", htmlCoverReport)

		if err := coverToHTML(); err != nil {
			return fmt.Errorf("failed to write HTML code coverage report: %w", err)
		}
	}

	// Return an error indicating that testing failed.
	if goTestErr != nil {
		fmt.Println(">> go test:", params.TestName, "Test Failed")
		return fmt.Errorf("go test returned a non-zero value: %w", goTestErr)
	}

	fmt.Println(">> go test:", params.TestName, "Test Passed")
	return nil
}

func makeCommand(ctx context.Context, env map[string]string, cmd string, args ...string) *exec.Cmd {
	c := exec.CommandContext(ctx, cmd, args...)
	c.Env = os.Environ()
	for k, v := range env {
		c.Env = append(c.Env, k+"="+v)
	}
	c.Stdout = io.Discard
	if mg.Verbose() {
		c.Stdout = os.Stdout
	}
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	log.Println("exec:", cmd, strings.Join(args, " "))
	fmt.Println("exec:", cmd, strings.Join(args, " "))
	return c
}

// BuildSystemTestBinary runs BuildSystemTestGoBinary with default values.
func BuildSystemTestBinary() error {
	return BuildSystemTestGoBinary(DefaultTestBinaryArgs())
}

// BuildSystemTestGoBinary build a binary for testing that is instrumented for
// testing and measuring code coverage. The binary is only instrumented for
// coverage when TEST_COVERAGE=true (default is false).
func BuildSystemTestGoBinary(binArgs TestBinaryArgs) error {
	args := []string{
		"test", "-c",
		"-o", binArgs.Name + ".test",
	}

	if DevBuild {
		// Disable optimizations (-N) and inlining (-l) for debugging.
		args = append(args, `-gcflags=all=-N -l`)
	}

	if TestCoverage {
		args = append(args, "-coverpkg", "./...")
	}
	args = append(args, binArgs.ExtraFlags...)
	if len(binArgs.InputFiles) > 0 {
		args = append(args, binArgs.InputFiles...)
	}

	start := time.Now()
	defer func() {
		log.Printf("BuildSystemTestGoBinary (go %v) took %v.", strings.Join(args, " "), time.Since(start))
	}()
	return sh.RunV("go", args...)
}
