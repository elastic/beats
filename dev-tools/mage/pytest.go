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
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	libbeatRequirements = "{{ elastic_beats_dir}}/libbeat/tests/system/requirements.txt"
)

var (
	// VirtualenvReqs specifies a list of virtualenv requirements files to be
	// used when calling PythonVirtualenv(). It defaults to the libbeat
	// requirements.txt file.
	VirtualenvReqs = []string{
		libbeatRequirements,
	}

	pythonVirtualenvDir  string // Location of python virtualenv (lazily set).
	pythonVirtualenvLock sync.Mutex

	// More globs may be needed in the future if tests are added in more places.
	nosetestsTestFiles = []string{
		"tests/system/test_*.py",
		"module/*/test_*.py",
		"module/*/*/test_*.py",
	}
)

// PythonTestArgs are the arguments used for the "python*Test" targets and they
// define how "nosetests" is invoked.
type PythonTestArgs struct {
	TestName            string            // Test name used in logging.
	Env                 map[string]string // Env vars to add to the current env.
	XUnitReportFile     string            // File to write the XUnit XML test report to.
	CoverageProfileFile string            // Test coverage profile file.
}

func makePythonTestArgs(name string) PythonTestArgs {
	fileName := fmt.Sprintf("build/TEST-python-%s", strings.Replace(strings.ToLower(name), " ", "_", -1))

	params := PythonTestArgs{
		TestName:        name,
		Env:             map[string]string{},
		XUnitReportFile: fileName + ".xml",
	}
	if TestCoverage {
		params.CoverageProfileFile = fileName + ".cov"
	}
	return params
}

// DefaultPythonTestUnitArgs returns a default set of arguments for running
// all unit tests.
func DefaultPythonTestUnitArgs() PythonTestArgs { return makePythonTestArgs("Unit") }

// DefaultPythonTestIntegrationArgs returns a default set of arguments for
// running all integration tests. Integration tests are made conditional by
// checking for INTEGRATION_TEST=1 in the test code.
func DefaultPythonTestIntegrationArgs() PythonTestArgs { return makePythonTestArgs("Integration") }

// PythonNoseTest invokes "nosetests" via a Python virtualenv.
func PythonNoseTest(params PythonTestArgs) error {
	fmt.Println(">> python test:", params.TestName, "Testing")

	ve, err := PythonVirtualenv()
	if err != nil {
		return err
	}

	nosetestsEnv := map[string]string{
		// activate sets this. Not sure if it's ever needed.
		"VIRTUAL_ENV": ve,
	}
	if IsInIntegTestEnv() {
		nosetestsEnv["INTEGRATION_TESTS"] = "1"
	}
	for k, v := range params.Env {
		nosetestsEnv[k] = v
	}

	nosetestsOptions := []string{
		"--process-timeout=90",
		"--with-timer",
	}
	if mg.Verbose() {
		nosetestsOptions = append(nosetestsOptions, "-v")
	}
	if params.XUnitReportFile != "" {
		nosetestsOptions = append(nosetestsOptions,
			"--with-xunit",
			"--xunit-file="+createDir(params.XUnitReportFile),
		)
	}

	testFiles, err := FindFiles(nosetestsTestFiles...)
	if err != nil {
		return err
	}
	if len(testFiles) == 0 {
		fmt.Println(">> python test:", params.TestName, "Testing - No tests found.")
		return nil
	}

	// We check both the VE and the normal PATH because on Windows if the
	// requirements are met by the globally installed package they are not
	// installed to the VE.
	nosetestsPath, err := LookVirtualenvPath(ve, "nosetests")
	if err != nil {
		return err
	}

	defer fmt.Println(">> python test:", params.TestName, "Testing Complete")
	return sh.RunWith(nosetestsEnv, nosetestsPath, append(nosetestsOptions, testFiles...)...)

	// TODO: Aggregate all the individual code coverage reports and generate
	// and HTML report.
}

// PythonVirtualenv constructs a virtualenv that contains the given modules as
// defined in the requirements file pointed to by requirementsTxt. It returns
// the path to the virutalenv.
func PythonVirtualenv() (string, error) {
	pythonVirtualenvLock.Lock()
	defer pythonVirtualenvLock.Unlock()

	// Determine the location of the virtualenv.
	ve, err := pythonVirtualenvPath()
	if err != nil {
		return "", err
	}

	reqs := expandVirtualenvReqs()

	// Only execute if requirements.txt is newer than the virtualenv activate
	// script.
	activate := virtualenvPath(ve, "activate")
	if IsUpToDate(activate, reqs...) {
		return pythonVirtualenvDir, nil
	}

	// If set use PYTHON_EXE env var as the python interpreter.
	var args []string
	if pythonExe := os.Getenv("PYTHON_EXE"); pythonExe != "" {
		args = append(args, "-p", pythonExe)
	}
	args = append(args, ve)

	// Execute virtualenv.
	if _, err := os.Stat(ve); err != nil {
		// Run virtualenv if the dir does not exist.
		if err := sh.Run("virtualenv", args...); err != nil {
			return "", err
		}
	}

	// activate sets this. Not sure if it's ever needed.
	env := map[string]string{
		"VIRTUAL_ENV": ve,
	}

	pip := virtualenvPath(ve, "pip")
	args = []string{"install"}
	if !mg.Verbose() {
		args = append(args, "--quiet")
	}
	for _, req := range reqs {
		args = append(args, "-Ur", req)
	}

	// Execute pip to install the dependencies.
	if err := sh.RunWith(env, pip, args...); err != nil {
		return "", err
	}

	// Touch activate script.
	mtime := time.Now()
	if err := os.Chtimes(activate, mtime, mtime); err != nil {
		log.Fatal(err)
	}

	return ve, nil
}

// pythonVirtualenvPath determines the location of the Python virtualenv.
func pythonVirtualenvPath() (string, error) {
	if pythonVirtualenvDir != "" {
		return pythonVirtualenvDir, nil
	}

	// PYTHON_ENV can override the default location. This is used by CI to
	// shorten the overall shebang interpreter path below the path length limits.
	pythonVirtualenvDir = os.Getenv("PYTHON_ENV")
	if pythonVirtualenvDir == "" {
		info, err := GetProjectRepoInfo()
		if err != nil {
			return "", err
		}

		pythonVirtualenvDir = info.RootDir
	}
	pythonVirtualenvDir = filepath.Join(pythonVirtualenvDir, "build/ve")

	// Use OS and docker specific virtualenv's because the interpreter in
	// scripts is different.
	if IsInIntegTestEnv() {
		pythonVirtualenvDir = filepath.Join(pythonVirtualenvDir, "docker")
	} else {
		pythonVirtualenvDir = filepath.Join(pythonVirtualenvDir, runtime.GOOS)
	}

	return pythonVirtualenvDir, nil
}

// virtualenvPath builds the path to a binary (in the OS specific binary path).
func virtualenvPath(ve string, parts ...string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(append([]string{ve, "Scripts"}, parts...)...)
	}
	return filepath.Join(append([]string{ve, "bin"}, parts...)...)
}

// LookVirtualenvPath looks for an executable in the path and it includes the
// virtualenv in the search.
func LookVirtualenvPath(ve, file string) (string, error) {
	// This is kind of unsafe w.r.t. concurrent execs because they could end
	// up with different PATHs. But it allows us to search the VE path without
	// having to re-implement the exec.LookPath logic. And does not require us
	// to "deactivate" the virtualenv because we never activated it.
	path := os.Getenv("PATH")
	os.Setenv("PATH", virtualenvPath(ve)+string(filepath.ListSeparator)+path)
	defer os.Setenv("PATH", path)

	return exec.LookPath(file)
}

func expandVirtualenvReqs() []string {
	out := make([]string, 0, len(VirtualenvReqs))
	for _, path := range VirtualenvReqs {
		out = append(out, MustExpand(path))
	}
	return out
}
