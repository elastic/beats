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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	// BEATS_DOCKER_INTEGRATION_TEST_ENV is used to indicate that we are inside
	// of the Docker integration test environment (e.g. in a container).
	beatsDockerIntegrationTestEnvVar = "BEATS_DOCKER_INTEGRATION_TEST_ENV"
)

var (
	integTestUseCount     int32      // Reference count for the integ test env.
	integTestUseCountLock sync.Mutex // Lock to guard integTestUseCount.

	integTestLock sync.Mutex // Only allow one integration test at a time.

	integTestBuildImagesOnce sync.Once // Build images one time for all integ testing.
)

// Integration Test Configuration
var (
	// StackEnvironment specifies what testing environment
	// to use (like snapshot (default), latest, 5x). Formerly known as
	// TESTING_ENVIRONMENT.
	StackEnvironment = EnvOr("STACK_ENVIRONMENT", "snapshot")
)

// AddIntegTestUsage increments the use count for the integration test
// environment and prevents it from being stopped until the last call to
// StopIntegTestEnv(). You should also pair this with
// 'defer StopIntegTestEnv()'.
//
// This allows for the same environment to be reused by multiple tests (like
// both Go and Python) without tearing it down in between runs.
func AddIntegTestUsage() {
	if IsInIntegTestEnv() {
		return
	}

	integTestUseCountLock.Lock()
	defer integTestUseCountLock.Unlock()
	integTestUseCount++
}

// StopIntegTestEnv will stop and removing the integration test environment
// (e.g. docker-compose rm --stop --force) when there are no more users
// of the environment.
func StopIntegTestEnv() error {
	if IsInIntegTestEnv() {
		return nil
	}

	integTestUseCountLock.Lock()
	defer integTestUseCountLock.Unlock()
	if integTestUseCount == 0 {
		panic("integTestUseCount is 0. Did you call AddIntegTestUsage()?")
	}

	integTestUseCount--
	if integTestUseCount > 0 {
		return nil
	}

	if err := haveIntegTestEnvRequirements(); err != nil {
		// Ignore error because it will be logged by RunIntegTest.
		return nil
	}

	if _, skip := skipIntegTest(); skip {
		return nil
	}

	composeEnv, err := integTestDockerComposeEnvVars()
	if err != nil {
		return err
	}

	// Stop docker-compose when reference count hits 0.
	fmt.Println(">> Stopping Docker test environment...")

	// Docker-compose rm is noisy. So only pass through stderr when in verbose.
	out := ioutil.Discard
	if mg.Verbose() {
		out = os.Stderr
	}

	_, err = sh.Exec(
		composeEnv,
		ioutil.Discard,
		out,
		"docker-compose",
		"-p", dockerComposeProjectName(),
		"rm", "--stop", "--force",
	)
	return err
}

// RunIntegTest executes the given target inside the integration testing
// environment (Docker).
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
// Use STACK_ENVIRONMENT=env to specify what testing environment
// to use (like snapshot (default), latest, 5x).
//
// Always use this with AddIntegTestUsage() and defer StopIntegTestEnv().
func RunIntegTest(mageTarget string, test func() error, passThroughEnvVars ...string) error {
	if reason, skip := skipIntegTest(); skip {
		fmt.Printf(">> %v: Skipping because %v\n", mageTarget, reason)
		return nil
	}

	AddIntegTestUsage()
	defer StopIntegTestEnv()

	env := []string{
		"TEST_COVERAGE",
		"RACE_DETECTOR",
	}
	env = append(env, passThroughEnvVars...)
	return runInIntegTestEnv(mageTarget, test, env...)
}

func runInIntegTestEnv(mageTarget string, test func() error, passThroughEnvVars ...string) error {
	if IsInIntegTestEnv() {
		// Fix file permissions after test is done writing files as root.
		if runtime.GOOS != "windows" {
			defer DockerChown(".")
		}
		return test()
	}

	var err error
	integTestBuildImagesOnce.Do(func() { err = dockerComposeBuildImages() })
	if err != nil {
		return err
	}

	// Test that we actually have Docker and docker-compose.
	if err := haveIntegTestEnvRequirements(); err != nil {
		return errors.Wrapf(err, "failed to run %v target in integration environment", mageTarget)
	}

	// Pre-build a mage binary to execute inside docker so that we don't need to
	// have mage installed inside the container.
	mg.Deps(buildMage)

	// Determine the path to use inside the container.
	repo, err := GetProjectRepoInfo()
	if err != nil {
		return err
	}
	magePath := filepath.Join("/go/src", repo.ImportPath, "build/mage-linux-amd64")

	// Build docker-compose args.
	args := []string{"-p", dockerComposeProjectName(), "run",
		"-e", "DOCKER_COMPOSE_PROJECT_NAME=" + dockerComposeProjectName(),
		// Disable strict.perms because we moust host dirs inside containers
		// and the UID/GID won't meet the strict requirements.
		"-e", "BEAT_STRICT_PERMS=false",
		// compose.EnsureUp needs to know the environment type.
		"-e", "STACK_ENVIRONMENT=" + StackEnvironment,
		"-e", "TESTING_ENVIRONMENT=" + StackEnvironment,
	}
	args, err = addUidGidEnvArgs(args)
	if err != nil {
		return err
	}
	for _, envVar := range passThroughEnvVars {
		args = append(args, "-e", envVar+"="+os.Getenv(envVar))
	}
	if mg.Verbose() {
		args = append(args, "-e", "MAGEFILE_VERBOSE=1")
	}
	args = append(args,
		"-e", beatsDockerIntegrationTestEnvVar+"=true",
		"beat", // Docker compose container name.
		magePath,
		mageTarget,
	)

	composeEnv, err := integTestDockerComposeEnvVars()
	if err != nil {
		return err
	}

	// Only allow one usage at a time.
	integTestLock.Lock()
	defer integTestLock.Unlock()

	_, err = sh.Exec(
		composeEnv,
		os.Stdout,
		os.Stderr,
		"docker-compose",
		args...,
	)
	return err
}

// IsInIntegTestEnv return true if executing inside the integration test
// environment.
func IsInIntegTestEnv() bool {
	_, found := os.LookupEnv(beatsDockerIntegrationTestEnvVar)
	return found
}

func haveIntegTestEnvRequirements() error {
	if err := HaveDockerCompose(); err != nil {
		return err
	}
	if err := HaveDocker(); err != nil {
		return err
	}
	return nil
}

// skipIntegTest returns true if integ tests should be skipped.
func skipIntegTest() (reason string, skip bool) {
	if IsInIntegTestEnv() {
		return "", false
	}

	// Honor the TEST_ENVIRONMENT value if set.
	if testEnvVar, isSet := os.LookupEnv("TEST_ENVIRONMENT"); isSet {
		enabled, err := strconv.ParseBool(testEnvVar)
		if err != nil {
			panic(errors.Wrap(err, "failed to parse TEST_ENVIRONMENT value"))
		}
		return "TEST_ENVIRONMENT=" + testEnvVar, !enabled
	}

	// Otherwise skip if we don't have all the right dependencies.
	if err := haveIntegTestEnvRequirements(); err != nil {
		// Skip if we don't meet the requirements.
		log.Println("Skipping integ test because:", err)
		return "docker is not available", true
	}

	return "", false
}

// integTestDockerComposeEnvVars returns the environment variables used for
// executing docker-compose (not the variables passed into the containers).
// docker-compose uses these when evaluating docker-compose.yml files.
func integTestDockerComposeEnvVars() (map[string]string, error) {
	esBeatsDir, err := ElasticBeatsDir()
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"ES_BEATS":          esBeatsDir,
		"STACK_ENVIRONMENT": StackEnvironment,
		// Deprecated use STACK_ENVIRONMENT instead (it's more descriptive).
		"TESTING_ENVIRONMENT": StackEnvironment,
	}, nil
}

// dockerComposeProjectName returns the project name to use with docker-compose.
// It is passed to docker-compose using the `-p` flag. And is passed to our
// Go and Python testing libraries through the DOCKER_COMPOSE_PROJECT_NAME
// environment variable.
func dockerComposeProjectName() string {
	commit, err := CommitHash()
	if err != nil {
		panic(errors.Wrap(err, "failed to construct docker compose project name"))
	}

	version, err := BeatQualifiedVersion()
	if err != nil {
		panic(errors.Wrap(err, "failed to construct docker compose project name"))
	}
	version = strings.NewReplacer(".", "_").Replace(version)

	projectName := "{{.BeatName}}_{{.Version}}_{{.ShortCommit}}-{{.StackEnvironment}}"
	projectName = MustExpand(projectName, map[string]interface{}{
		"StackEnvironment": StackEnvironment,
		"ShortCommit":      commit[:10],
		"Version":          version,
	})
	return projectName
}

// dockerComposeBuildImages builds all images in the docker-compose.yml file.
func dockerComposeBuildImages() error {
	fmt.Println(">> Building docker images")

	composeEnv, err := integTestDockerComposeEnvVars()
	if err != nil {
		return err
	}

	args := []string{"-p", dockerComposeProjectName(), "build", "--pull", "--force-rm"}
	if _, noCache := os.LookupEnv("DOCKER_NOCACHE"); noCache {
		args = append(args, "--no-cache")
	}

	out := ioutil.Discard
	if mg.Verbose() {
		out = os.Stderr
	}

	_, err = sh.Exec(
		composeEnv,
		out,
		os.Stderr,
		"docker-compose", args...,
	)
	return err
}
