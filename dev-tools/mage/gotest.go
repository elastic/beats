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
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/jstemmer/go-junit-report/formatter"
	"github.com/jstemmer/go-junit-report/parser"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"
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
}

// TestBinaryArgs are the arguments used when building binary for testing.
type TestBinaryArgs struct {
	Name       string // Name of the binary to build
	InputFiles []string
}

func makeGoTestArgs(name string) GoTestArgs {
	fileName := fmt.Sprintf("build/TEST-go-%s", strings.Replace(strings.ToLower(name), " ", "_", -1))
	params := GoTestArgs{
		TestName:        name,
		Race:            RaceDetector,
		Packages:        []string{"./..."},
		OutputFile:      fileName + ".out",
		JUnitReportFile: fileName + ".xml",
	}
	if TestCoverage {
		params.CoverageProfileFile = fileName + ".cov"
	}
	return params
}

// DefaultGoTestUnitArgs returns a default set of arguments for running
// all unit tests. We tag unit test files with '!integration'.
func DefaultGoTestUnitArgs() GoTestArgs { return makeGoTestArgs("Unit") }

// DefaultGoTestIntegrationArgs returns a default set of arguments for running
// all integration tests. We tag integration test files with 'integration'.
func DefaultGoTestIntegrationArgs() GoTestArgs {
	args := makeGoTestArgs("Integration")
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

// GoTest invokes "go test" and reports the results to stdout. It returns an
// error if there was any failure executing the tests or if there were any
// test failures.
func GoTest(ctx context.Context, params GoTestArgs) error {
	fmt.Println(">> go test:", params.TestName, "Testing")

	// Build args list to Go.
	args := []string{"test", "-v"}
	if params.Race {
		args = append(args, "-race")
	}
	if len(params.Tags) > 0 {
		args = append(args, "-tags", strings.Join(params.Tags, " "))
	}
	if params.CoverageProfileFile != "" {
		params.CoverageProfileFile = createDir(filepath.Clean(params.CoverageProfileFile))
		args = append(args,
			"-covermode=atomic",
			"-coverprofile="+params.CoverageProfileFile,
		)
	}
	args = append(args, params.ExtraFlags...)
	args = append(args, params.Packages...)

	goTest := makeCommand(ctx, params.Env, "go", args...)

	// Wire up the outputs.
	bufferOutput := new(bytes.Buffer)
	outputs := []io.Writer{bufferOutput}
	if mg.Verbose() {
		outputs = append(outputs, os.Stdout)
	}
	if params.OutputFile != "" {
		fileOutput, err := os.Create(createDir(params.OutputFile))
		if err != nil {
			return errors.Wrap(err, "failed to create go test output file")
		}
		defer fileOutput.Close()
		outputs = append(outputs, fileOutput)
	}
	output := io.MultiWriter(outputs...)
	goTest.Stdout = output
	goTest.Stderr = output

	// Execute 'go test' and measure duration.
	start := time.Now()
	err := goTest.Run()
	duration := time.Since(start)
	var goTestErr *exec.ExitError
	if err != nil {
		// Command ran.
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			return errors.Wrap(err, "failed to execute go")
		}

		// Command ran but failed. Process the output.
		goTestErr = exitErr
	}

	// Parse the verbose test output.
	report, err := parser.Parse(bytes.NewBuffer(bufferOutput.Bytes()), BeatName)
	if err != nil {
		return errors.Wrap(err, "failed to parse go test output")
	}
	if goTestErr != nil && len(report.Packages) == 0 {
		// No packages were tested. Probably the code didn't compile.
		fmt.Println(bytes.NewBuffer(bufferOutput.Bytes()).String())
		return errors.Wrap(goTestErr, "go test returned a non-zero value")
	}

	// Generate a JUnit XML report.
	if params.JUnitReportFile != "" {
		junitReport, err := os.Create(createDir(params.JUnitReportFile))
		if err != nil {
			return errors.Wrap(err, "failed to create junit report")
		}
		defer junitReport.Close()

		if err = formatter.JUnitReportXML(report, false, runtime.Version(), junitReport); err != nil {
			return errors.Wrap(err, "failed to write junit report")
		}
	}

	// Generate a HTML code coverage report.
	var htmlCoverReport string
	if params.CoverageProfileFile != "" {
		htmlCoverReport = strings.TrimSuffix(params.CoverageProfileFile,
			filepath.Ext(params.CoverageProfileFile)) + ".html"
		coverToHTML := sh.RunCmd("go", "tool", "cover",
			"-html="+params.CoverageProfileFile,
			"-o", htmlCoverReport)
		if err = coverToHTML(); err != nil {
			return errors.Wrap(err, "failed to write HTML code coverage report")
		}
	}

	// Summarize the results and log to stdout.
	summary, err := NewGoTestSummary(duration, report, map[string]string{
		"Output File":     params.OutputFile,
		"JUnit Report":    params.JUnitReportFile,
		"Coverage Report": htmlCoverReport,
	})
	if err != nil {
		return err
	}
	if !mg.Verbose() && summary.Fail > 0 {
		fmt.Println(summary.Failures())
	}
	fmt.Println(summary.String())

	// Return an error indicating that testing failed.
	if summary.Fail > 0 || goTestErr != nil {
		fmt.Println(">> go test:", params.TestName, "Test Failed")
		if summary.Fail > 0 {
			return errors.Errorf("go test failed: %d test failures", summary.Fail)
		}

		return errors.Wrap(goTestErr, "go test returned a non-zero value")
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
	c.Stdout = ioutil.Discard
	if mg.Verbose() {
		c.Stdout = os.Stdout
	}
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	log.Println("exec:", cmd, strings.Join(args, " "))
	return c
}

// GoTestSummary is a summary of test results.
type GoTestSummary struct {
	*parser.Report               // Report generated by parsing test output.
	Pass           int           // Number of passing tests.
	Fail           int           // Number of failed tests.
	Skip           int           // Number of skipped tests.
	Packages       int           // Number of packages tested.
	Duration       time.Duration // Total go test running duration.
	Files          map[string]string
}

// NewGoTestSummary builds a new GoTestSummary. It returns an error if it cannot
// resolve the absolute paths to the given files.
func NewGoTestSummary(d time.Duration, r *parser.Report, outputFiles map[string]string) (*GoTestSummary, error) {
	files := map[string]string{}
	for name, file := range outputFiles {
		if file == "" {
			continue
		}
		absFile, err := filepath.Abs(file)
		if err != nil {
			return nil, errors.Wrapf(err, "failed resolving absolute path for %v", file)
		}
		files[name+":"] = absFile
	}

	summary := &GoTestSummary{
		Report:   r,
		Duration: d,
		Packages: len(r.Packages),
		Files:    files,
	}

	for _, pkg := range r.Packages {
		for _, t := range pkg.Tests {
			switch t.Result {
			case parser.PASS:
				summary.Pass++
			case parser.FAIL:
				summary.Fail++
			case parser.SKIP:
				summary.Skip++
			default:
				return nil, errors.Errorf("Unknown test result value: %v", t.Result)
			}
		}
	}

	return summary, nil
}

// Failures returns a string containing the list of failed test cases and their
// output.
func (s *GoTestSummary) Failures() string {
	b := new(strings.Builder)

	if s.Fail > 0 {
		fmt.Fprintln(b, "FAILURES:")
		for _, pkg := range s.Report.Packages {
			for _, t := range pkg.Tests {
				if t.Result != parser.FAIL {
					continue
				}
				fmt.Fprintln(b, "Package:", pkg.Name)
				fmt.Fprintln(b, "Test:   ", t.Name)
				for _, line := range t.Output {
					if strings.TrimSpace(line) != "" {
						fmt.Fprintln(b, line)
					}
				}
				fmt.Fprintln(b, "----")
			}
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

// String returns a summary of the testing results (number of fail/pass/skip,
// test duration, number packages, output files).
func (s *GoTestSummary) String() string {
	b := new(strings.Builder)

	fmt.Fprintln(b, "SUMMARY:")
	fmt.Fprintln(b, "  Fail:    ", s.Fail)
	fmt.Fprintln(b, "  Skip:    ", s.Skip)
	fmt.Fprintln(b, "  Pass:    ", s.Pass)
	fmt.Fprintln(b, "  Packages:", len(s.Report.Packages))
	fmt.Fprintln(b, "  Duration:", s.Duration)

	// Sort the list of files and compute the column width.
	var names []string
	var nameWidth int
	for name := range s.Files {
		if len(name) > nameWidth {
			nameWidth = len(name)
		}
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Fprintf(b, "  %-*s %s\n", nameWidth, name, s.Files[name])
	}

	return strings.TrimRight(b.String(), "\n")
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
	if TestCoverage {
		args = append(args, "-coverpkg", "./...")
	}
	if len(binArgs.InputFiles) > 0 {
		args = append(args, binArgs.InputFiles...)
	}
	return sh.RunV("go", args...)
}
