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

// WINDOWS USERS:
// The python installer does not create a python3 alias like it does on other
// platforms. So do verify the version with python.exe --version.
//
// Setting up a python virtual environment on a network drive does not work
// well. So if this applies to your development environment set PYTHON_ENV
// to point to somewhere on C:\.

const (
	libbeatRequirements    = "{{ elastic_beats_dir}}/libbeat/tests/system/requirements.txt"
	aixLibbeatRequirements = "{{ elastic_beats_dir}}/libbeat/tests/system/requirements_aix.txt"
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
	pythonTestFiles = []string{
		"tests/system/test_*.py",
		"module/*/test_*.py",
		"module/*/*/test_*.py",
	}

	// pythonExe points to the python executable to use. The PYTHON_EXE
	// environment can be used to modify the executable used.
	// On Windows this defaults to python and on all other platforms this
	// defaults to python3.
	pythonExe = EnvOr("PYTHON_EXE", "python3")
)

func init() {
	// The python installer for Windows does not setup a python3 alias.
	if runtime.GOOS == "windows" {
		pythonExe = EnvOr("PYTHON_EXE", "python")
	}
}

// PythonTestArgs are the arguments used for the "python*Test" targets and they
// define how python tests are invoked.
type PythonTestArgs struct {
	TestName            string            // Test name used in logging.
	Env                 map[string]string // Env vars to add to the current env.
	Files               []string          // Globs used to find tests.
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

// PythonTest executes python tests via a Python virtualenv.
func PythonTest(params PythonTestArgs) error {
	fmt.Println(">> python test:", params.TestName, "Testing")

	ve, err := PythonVirtualenv()
	if err != nil {
		return err
	}

	pytestEnv := map[string]string{
		// activate sets this. Not sure if it's ever needed.
		"VIRTUAL_ENV": ve,
	}
	if IsInIntegTestEnv() {
		pytestEnv["INTEGRATION_TESTS"] = "1"
	}
	for k, v := range params.Env {
		pytestEnv[k] = v
	}

	pytestOptions := []string{
		"--timeout=90",
		"--durations=20",
		// Enable -x to stop at the first failing test
		// "-x",
		// Enable --tb=long to produce long tracebacks
		//"--tb=long",
		// Enable -v to produce verbose output
		//"-v",
		// Don't capture test output
		//"-s",
	}
	if mg.Verbose() {
		pytestOptions = append(pytestOptions, "-v")
	}
	if params.XUnitReportFile != "" {
		pytestOptions = append(pytestOptions,
			"--junit-xml="+createDir(params.XUnitReportFile),
		)
	}

	files := pythonTestFiles
	if len(params.Files) > 0 {
		files = params.Files
	}
	testFiles, err := FindFiles(files...)
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
	pytestPath, err := LookVirtualenvPath(ve, "pytest")
	if err != nil {
		return err
	}

	defer fmt.Println(">> python test:", params.TestName, "Testing Complete")
	_, err = sh.Exec(pytestEnv, os.Stdout, os.Stderr, pytestPath, append(pytestOptions, testFiles...)...)
	return err

	// TODO: Aggregate all the individual code coverage reports and generate
	// and HTML report.
}

// PythonTestForModule executes python system tests for modules.
//
// Use `MODULE=module` to run only tests for `module`.
func PythonTestForModule(params PythonTestArgs) error {
	if module := EnvOr("MODULE", ""); module != "" {
		fmt.Println(">> Single module selected for testing: ", module)
		params.Files = []string{
			fmt.Sprintf("module/%s/test_*.py", module),
			fmt.Sprintf("module/%s/*/test_*.py", module),

			// Run always the base tests, that include tests for module dashboards.
			"tests/system/test*_base.py",
		}
		fmt.Println("Test files: ", params.Files)
		params.TestName += "-" + module
	} else {
		fmt.Println(">> Running tests for all modules, you can use MODULE=foo to scope it down to a single module...")
	}
	return PythonTest(params)
}

// PythonVirtualenv constructs a virtualenv that contains the given modules as
// defined in the requirements file pointed to by requirementsTxt. It returns
// the path to the virtualenv.
func PythonVirtualenv() (string, error) {
	pythonVirtualenvLock.Lock()
	defer pythonVirtualenvLock.Unlock()

	// Certain docker requirements simply won't build on AIX
	// Skipping them here will obviously break the components that require docker-compose,
	// But at least the components that don't require it will still run
	if runtime.GOOS == "aix" {
		VirtualenvReqs[0] = aixLibbeatRequirements
	}

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

	// Create a virtual environment only if the dir does not exist.
	if _, err := os.Stat(ve); err != nil {
		if err := sh.Run(pythonExe, "-m", "venv", ve); err != nil {
			return "", err
		}
	}

	// activate sets this. Not sure if it's ever needed.
	env := map[string]string{
		"VIRTUAL_ENV": ve,
	}

	pip := virtualenvPath(ve, "pip")
	pipUpgrade := func(pkg string) error {
		return sh.RunWith(env, pip, "install", "-U", pkg)
	}

	// Ensure we are using the latest pip version.
	if err = pipUpgrade("pip"); err != nil {
		fmt.Printf("warn: failed to upgrade pip (ignoring): %v", err)
	}

	// First ensure that wheel is installed so that bdists build cleanly.
	if err = pipUpgrade("wheel"); err != nil {
		return "", err
	}

	// Execute pip to install the dependencies.
	args := []string{"install"}
	if !mg.Verbose() {
		args = append(args, "--quiet")
	}
	for _, req := range reqs {
		args = append(args, "-Ur", req)
	}
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
