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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/joeshaw/multierror"
	"github.com/magefile/mage/mg"
)

const (
	// BEATS_INSIDE_INTEGRATION_TEST_ENV is used to indicate that we are inside
	// of the integration test environment.
	insideIntegrationTestEnvVar = "BEATS_INSIDE_INTEGRATION_TEST_ENV"
)

var (
	globalIntegrationTesters        map[string]IntegrationTester
	globalIntegrationTestSetupSteps IntegrationTestSteps

	defaultPassthroughEnvVars = []string{
		"TEST_COVERAGE",
		"RACE_DETECTOR",
		"TEST_TAGS",
		"PYTHON_EXE",
		"MODULE",
		"KUBECONFIG",
		"KUBE_CONFIG",
	}
)

// RegisterIntegrationTester registers a integration tester.
func RegisterIntegrationTester(tester IntegrationTester) {
	if globalIntegrationTesters == nil {
		globalIntegrationTesters = make(map[string]IntegrationTester)
	}
	globalIntegrationTesters[tester.Name()] = tester
}

// RegisterIntegrationTestSetupStep registers a integration step.
func RegisterIntegrationTestSetupStep(step IntegrationTestSetupStep) {
	globalIntegrationTestSetupSteps = append(globalIntegrationTestSetupSteps, step)
}

// IntegrationTestSetupStep is interface used by a step in the integration setup
// chain. Example could be: Terraform -> Kind -> Kubernetes (IntegrationTester).
type IntegrationTestSetupStep interface {
	// Name is the name of the step.
	Name() string
	// Use returns true in the case that the step should be used. Not called
	// when a step is defined as a dependency of a tester.
	Use(dir string) (bool, error)
	// Setup sets up the environment for the integration test.
	Setup(env map[string]string) error
	// Teardown brings down the environment for the integration test.
	Teardown(env map[string]string) error
}

// IntegrationTestSteps wraps all the steps and completes the in the order added.
type IntegrationTestSteps []IntegrationTestSetupStep

// Name is the name of the step.
func (steps IntegrationTestSteps) Name() string {
	return "IntegrationTestSteps"
}

// Setup calls Setup on each step in the order defined.
//
// In the case that Setup fails on a step, Teardown will be called on the previous
// successful steps.
func (steps IntegrationTestSteps) Setup(env map[string]string) error {
	for i, step := range steps {
		if mg.Verbose() {
			fmt.Printf("Setup %s...\n", step.Name())
		}
		if err := step.Setup(env); err != nil {
			prev := i - 1
			if prev >= 0 {
				// errors ignored
				_ = steps.teardownFrom(prev, env)
			}
			return fmt.Errorf("%s setup failed: %w", step.Name(), err)
		}
	}
	return nil
}

// Teardown calls Teardown in the reverse order defined.
//
// In the case a teardown step fails the error is recorded but the
// previous steps teardown is still called. This guarantees that teardown
// will always be called for each step.
func (steps IntegrationTestSteps) Teardown(env map[string]string) error {
	return steps.teardownFrom(len(steps)-1, env)
}

func (steps IntegrationTestSteps) teardownFrom(start int, env map[string]string) error {
	var errs multierror.Errors
	for i := start; i >= 0; i-- {
		if mg.Verbose() {
			fmt.Printf("Teardown %s...\n", steps[i].Name())
		}
		if err := steps[i].Teardown(env); err != nil {
			errs = append(errs, fmt.Errorf("%s teardown failed: %w", steps[i].Name(), err))
		}
	}
	return errs.Err()
}

// IntegrationTester is interface used by the actual test runner.
type IntegrationTester interface {
	// Name returns the name of the tester.
	Name() string
	// Use returns true in the case that the tester should be used.
	Use(dir string) (bool, error)
	// HasRequirements returns an error if requirements are missing.
	HasRequirements() error
	// Test performs executing the test inside the environment.
	Test(dir string, mageTarget string, env map[string]string) error
	// InsideTest performs the actual test on the inside of environment.
	InsideTest(test func() error) error
	// StepRequirements returns the steps this tester requires. These
	// are always placed before other autodiscover steps.
	StepRequirements() IntegrationTestSteps
}

// IntegrationRunner performs the running of the integration tests.
type IntegrationRunner struct {
	steps  IntegrationTestSteps
	tester IntegrationTester
	dir    string
	env    map[string]string
}

// IntegrationRunners is an array of multiple runners.
type IntegrationRunners []*IntegrationRunner

// NewIntegrationRunners returns the integration test runners discovered from the provided path.
func NewIntegrationRunners(path string, passInEnv map[string]string) (IntegrationRunners, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(cwd, path)

	// Load the overall steps to use (skipped inside of test environment, as they are never ran on the inside).
	// These steps are duplicated per scenario.
	if !IsInIntegTestEnv() {
		for _, step := range globalIntegrationTestSetupSteps {
			_, err := step.Use(dir)
			if err != nil {
				return nil, fmt.Errorf("%s step failed on Use: %w", step.Name(), err)
			}
		}
	}

	// Create the runners (can only be multiple).
	runners := make(IntegrationRunners, 0, len(globalIntegrationTesters))
	for _, t := range globalIntegrationTesters {
		use, err := t.Use(dir)
		if err != nil {
			return nil, fmt.Errorf("%s tester failed on Use: %w", t.Name(), err)
		}
		if !use {
			continue
		}
		runner, err := initRunner(t, dir, passInEnv)
		if err != nil {
			return nil, fmt.Errorf("initializing %s runner: %w", t.Name(), err)
		}
		runners = append(runners, runner)
	}
	// Keep support for modules that don't have a local environment defined at the module
	// level (system, stack and cloud modules by now)
	if len(runners) == 0 {
		if mg.Verbose() {
			fmt.Printf(">> No runner found in %s, using docker\n", path)
		}
		tester, ok := globalIntegrationTesters["docker"]
		if !ok {
			return nil, errors.New("docker integration test runner not registered")
		}
		runner, err := initRunner(tester, dir, passInEnv)
		if err != nil {
			return nil, fmt.Errorf("initializing docker runner: %w", err)
		}
		runners = append(runners, runner)
	}
	return runners, nil
}

// NewDockerIntegrationRunner returns an integration runner configured only for docker.
func NewDockerIntegrationRunner(passThroughEnvVars ...string) (*IntegrationRunner, error) {
	fmt.Printf("hi fae, NewDockerIntegrationRunner\n")
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	tester, ok := globalIntegrationTesters["docker"]
	if !ok {
		return nil, fmt.Errorf("docker integration test runner not registered")
	}
	passInEnv := make(map[string]string)
	if path, err := exec.LookPath("docker"); err == nil {
		dockerDir := filepath.Dir(path)
		passInEnv["PATH"] = dockerDir

	}
	return initRunner(tester, cwd, passInEnv, passThroughEnvVars...)
}

func initRunner(tester IntegrationTester, dir string, passInEnv map[string]string, passThroughEnvVars ...string) (*IntegrationRunner, error) {
	fmt.Printf("hi fae, initRunner\n")
	var runnerSteps IntegrationTestSteps
	requirements := tester.StepRequirements()
	if requirements != nil {
		fmt.Printf("hi fae, requirements: %v\n", requirements.Name())
		runnerSteps = append(runnerSteps, requirements...)
	}

	// Create the custom env for the runner.
	env := map[string]string{
		insideIntegrationTestEnvVar: "true",
		"GOFLAGS":                   "-mod=readonly",
	}
	for name, value := range passInEnv {
		env[name] = value
	}
	passThroughEnvs(env, passThroughEnvVars...)
	passThroughEnvs(env, defaultPassthroughEnvVars...)
	if mg.Verbose() {
		env["MAGEFILE_VERBOSE"] = "1"
	}

	runner := &IntegrationRunner{
		steps:  runnerSteps,
		tester: tester,
		dir:    dir,
		env:    env,
	}
	return runner, nil
}

// Test actually performs the test.
func (r *IntegrationRunner) Test(mageTarget string, test func() error) (err error) {
	// Inside the testing environment just run the test.
	if IsInIntegTestEnv() {
		err = r.tester.InsideTest(test)
		return
	}

	// Honor the TEST_ENVIRONMENT value if set.
	if testEnvVar, isSet := os.LookupEnv("TEST_ENVIRONMENT"); isSet {
		var enabled bool
		enabled, err = strconv.ParseBool(testEnvVar)
		if err != nil {
			err = fmt.Errorf("failed to parse TEST_ENVIRONMENT value: %w", err)
			return
		}
		if !enabled {
			err = fmt.Errorf("TEST_ENVIRONMENT=%s", testEnvVar)
			return
		}
	}

	// log missing requirements and do nothing
	err = r.tester.HasRequirements()
	if err != nil {
		// log error; and return (otherwise on machines without requirements it will mark the tests as failed)
		fmt.Printf("skipping test run with %s due to missing requirements: %s\n", r.tester.Name(), err)
		err = nil
		return
	}

	if err = r.steps.Setup(r.env); err != nil {
		return
	}

	// catch any panics to run teardown
	inTeardown := false
	defer func() {
		if recoverErr := recover(); recoverErr != nil {
			err = recoverErr.(error)
			if !inTeardown {
				// ignore errors
				_ = r.steps.Teardown(r.env)
			}
		}
	}()

	if mg.Verbose() {
		fmt.Printf(">> Running testing inside of %s...\n", r.tester.Name())
	}

	err = r.tester.Test(r.dir, mageTarget, r.env)

	if mg.Verbose() {
		fmt.Printf(">> Done running testing inside of %s...\n", r.tester.Name())
	}

	inTeardown = true
	if teardownErr := r.steps.Teardown(r.env); teardownErr != nil {
		if err == nil {
			// test didn't error, but teardown did
			err = teardownErr
		}
	}
	return err
}

// Test runs the test on each runner and collects the errors.
func (r IntegrationRunners) Test(mageTarget string, test func() error) error {
	var errs multierror.Errors
	for _, runner := range r {
		if err := runner.Test(mageTarget, test); err != nil {
			errs = append(errs, err)
		}
	}
	return errs.Err()
}

func passThroughEnvs(env map[string]string, passthrough ...string) {
	for _, envName := range passthrough {
		val, set := os.LookupEnv(envName)
		if set {
			env[envName] = val
		}
	}
}

// IsInIntegTestEnv return true if executing inside the integration test environment.
func IsInIntegTestEnv() bool {
	_, found := os.LookupEnv(insideIntegrationTestEnvVar)
	return found
}
