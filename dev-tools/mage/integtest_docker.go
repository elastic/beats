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
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var (
	// StackEnvironment specifies what testing environment
	// to use (like snapshot (default), latest, 5x). Formerly known as
	// TESTING_ENVIRONMENT.
	StackEnvironment = EnvOr("STACK_ENVIRONMENT", "snapshot")

	buildContainersOnce sync.Once
)

func init() {
	RegisterIntegrationTester(&DockerIntegrationTester{})
}

// DockerIntegrationTester is an integration tester that executes integration tests
// using docker-compose. The tests are run from inside a special beat container.
// Prefer using GoIntegTest and PythonIntegTest below which run the tests directly
// from the host system, avoiding the need to compile beats inside a test container.
type DockerIntegrationTester struct {
}

// Name returns docker name.
func (d *DockerIntegrationTester) Name() string {
	return "docker"
}

// Use determines if this tester should be used.
func (d *DockerIntegrationTester) Use(dir string) (bool, error) {
	dockerFile := filepath.Join(dir, "docker-compose.yml")
	if _, err := os.Stat(dockerFile); !os.IsNotExist(err) {
		return true, nil
	}
	return false, nil
}

// HasRequirements ensures that the required docker and docker-compose are installed.
func (d *DockerIntegrationTester) HasRequirements() error {
	fmt.Printf("hi fae, in DockerIntegrationTester.HasRequirements\n")
	if err := HaveDocker(); err != nil {
		return err
	}
	if err := HaveDockerCompose(); err != nil {
		return err
	}
	fmt.Printf("everything passed\n")
	os.Exit(1)
	return nil
}

// StepRequirements returns the steps required for this tester.
func (d *DockerIntegrationTester) StepRequirements() IntegrationTestSteps {
	return IntegrationTestSteps{&MageIntegrationTestStep{}}
}

// Test performs the tests with docker-compose. The compose file must define a "beat" container,
// containing the beats development environment. The tests are executed from within this container.
func (d *DockerIntegrationTester) Test(dir string, mageTarget string, env map[string]string) error {
	fmt.Printf("hi fae, DockerIntegrationTester.Test\n")
	var err error
	buildContainersOnce.Do(func() { err = BuildIntegTestContainers() })
	if err != nil {
		return err
	}

	// Determine the path to use inside the container.
	repo, err := GetProjectRepoInfo()
	if err != nil {
		return err
	}
	dockerRepoRoot := filepath.Join("/go/src", repo.CanonicalRootImportPath)
	dockerGoCache := filepath.Join(dockerRepoRoot, "build/docker-gocache")
	magePath := filepath.Join("/go/src", repo.CanonicalRootImportPath, repo.SubDir, "build/mage-linux-"+GOARCH)
	goPkgCache := filepath.Join(filepath.SplitList(build.Default.GOPATH)[0], "pkg/mod/cache/download")
	dockerGoPkgCache := "/gocache"

	// Execute the inside of docker-compose.
	args := []string{"-p", DockerComposeProjectName(), "run",
		"-e", "DOCKER_COMPOSE_PROJECT_NAME=" + DockerComposeProjectName(),
		// Disable strict.perms because we mount host dirs inside containers
		// and the UID/GID won't meet the strict requirements.
		"-e", "BEAT_STRICT_PERMS=false",
		// compose.EnsureUp needs to know the environment type.
		"-e", "STACK_ENVIRONMENT=" + StackEnvironment,
		"-e", "TESTING_ENVIRONMENT=" + StackEnvironment,
		"-e", "GOCACHE=" + dockerGoCache,
		// Use the host machine's pkg cache to minimize external downloads.
		"-v", goPkgCache + ":" + dockerGoPkgCache + ":ro",
		"-e", "GOPROXY=file://" + dockerGoPkgCache + ",direct",
	}
	args, err = addUidGidEnvArgs(args)
	if err != nil {
		return err
	}
	for envName, envVal := range env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", envName, envVal))
	}
	args = append(args,
		"beat", // Docker compose container name.
		magePath,
		mageTarget,
	)

	composeEnv, err := integTestDockerComposeEnvVars()
	if err != nil {
		return err
	}

	_, testErr := sh.Exec(
		composeEnv,
		os.Stdout,
		os.Stderr,
		"docker-compose",
		args...,
	)

	err = saveDockerComposeLogs(dir, mageTarget)
	if err != nil {
		// Just log the error, need to make sure the containers are stopped.
		fmt.Printf("Failed to save docker-compose logs: %s\n", err)
	}

	err = StopIntegTestContainers()
	if err != nil && testErr == nil {
		// Stopping containers failed but the test didn't
		return err
	}

	return testErr
}

// InsideTest performs the tests inside of environment.
func (d *DockerIntegrationTester) InsideTest(test func() error) error {
	// Fix file permissions after test is done writing files as root.
	if runtime.GOOS != "windows" {
		repo, err := GetProjectRepoInfo()
		if err != nil {
			return err
		}

		// Handle virtualenv and the current project dir.
		defer DockerChown(path.Join(repo.RootDir, "build"))
		defer DockerChown(".")
	}
	return test()
}

const dockerServiceHostname = "localhost"

// WithGoIntegTestHostEnv adds the integeration testing environment variables needed when running Go
// test from the host system with GoIntegTestFromHost().
func WithGoIntegTestHostEnv(env map[string]string) map[string]string {
	env["ES_HOST"] = dockerServiceHostname
	env["ES_USER"] = "beats"
	env["ES_PASS"] = "testing"
	env["ES_SUPERUSER_USER"] = "admin"
	env["ES_SUPERUSER_PASS"] = "testing"

	env["KIBANA_HOST"] = dockerServiceHostname
	env["KIBANA_USER"] = "beats"
	env["KIBANA_PASS"] = "testing"

	env["REDIS_HOST"] = dockerServiceHostname
	env["SREDIS_HOST"] = dockerServiceHostname
	env["LS_HOST"] = dockerServiceHostname

	// Allow connecting to older versions in tests. There can be a delay producing the snapshot
	// images for the next release after a feature freeze, which causes temporary test failures.
	env["TESTING_FILEBEAT_ALLOW_OLDER"] = "1"

	return env
}

// WithPythonIntegTestHostEnv adds the integeration testing environment variables needed when running
// pytest from the host system with PythonIntegTestFromHost().
func WithPythonIntegTestHostEnv(env map[string]string) map[string]string {
	env["INTEGRATION_TESTS"] = "1"
	env["MODULES_PATH"] = CWD("module")
	return WithGoIntegTestHostEnv(env)
}

// GoIntegTestFromHost starts docker-compose, waits for services to be healthy, and then runs "go test" on
// the host system with the arguments set to enable integration tests. The test results are printed
// to stdout and the container logs are saved in the build/system-test directory.
func GoIntegTestFromHost(ctx context.Context, params GoTestArgs) error {
	var err error
	buildContainersOnce.Do(func() { err = BuildIntegTestContainers() })
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting cwd: %w", err)
	}

	err = StartIntegTestContainers()
	if err != nil {
		return fmt.Errorf("starting containers: %w", err)
	}

	// Run Go test from the host machine. Do not immediately exit on error to allow cleanup to occur.
	testErr := GoTest(ctx, params)

	err = saveDockerComposeLogs(cwd, "goIntegTest")
	if err != nil {
		// Just log the error, need to make sure the containers are stopped.
		fmt.Printf("Failed to save docker-compose logs: %s\n", err)
	}

	err = StopIntegTestContainers()
	if err != nil && testErr == nil {
		// Stopping containers failed but the test didn't
		return err
	}

	return testErr
}

// PythonIntegTest starts docker-compose, waits for services to be healthy, and then runs "pytest" on
// the host system with the arguments set to enable integration tests. The test results are printed
// to stdout and the container logs are saved in the build/system-test directory.
func PythonIntegTestFromHost(params PythonTestArgs) error {
	var err error
	buildContainersOnce.Do(func() { err = BuildIntegTestContainers() })
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting cwd: %w", err)
	}

	err = StartIntegTestContainers()
	if err != nil {
		return fmt.Errorf("starting containers: %w", err)
	}

	// Run pytest from the host machine. Do not immediately exit on error to allow cleanup to occur.
	testErr := PythonTest(params)

	err = saveDockerComposeLogs(cwd, "pythonIntegTest")
	if err != nil {
		// Just log the error, need to make sure the containers are stopped.
		fmt.Printf("Failed to save docker-compose logs: %s\n", err)
	}

	err = StopIntegTestContainers()
	if err != nil && testErr == nil {
		// Stopping containers failed but the test didn't
		return err
	}

	return testErr
}

// dockerComposeBuildImages builds all images in the docker-compose.yml file.
func BuildIntegTestContainers() error {
	fmt.Println(">> Building docker images")

	composeEnv, err := integTestDockerComposeEnvVars()
	if err != nil {
		return err
	}

	args := []string{"-p", DockerComposeProjectName(), "build", "--force-rm"}
	if _, noCache := os.LookupEnv("DOCKER_NOCACHE"); noCache {
		args = append(args, "--no-cache")
	}

	if _, forcePull := os.LookupEnv("DOCKER_PULL"); forcePull {
		args = append(args, "--pull")
	}

	out := io.Discard
	if mg.Verbose() {
		out = os.Stderr
	}

	_, err = sh.Exec(
		composeEnv,
		out,
		os.Stderr,
		"docker-compose", args...,
	)

	// This sleep is to avoid hitting the docker build issues when resources are not available.
	if err != nil {
		fmt.Println(">> Building docker images again")
		time.Sleep(10 * time.Nanosecond)
		_, err = sh.Exec(
			composeEnv,
			out,
			os.Stderr,
			"docker-compose", args...,
		)
	}
	return err
}

func StartIntegTestContainers() error {
	// Start the docker-compose services and wait for them to become healthy.
	// Using --detach causes the command to exit successfully only if the proxy_dep for health
	// completed successfully.
	args := []string{"-p", DockerComposeProjectName(),
		"up",
		"--detach",
	}

	composeEnv, err := integTestDockerComposeEnvVars()
	if err != nil {
		return err
	}

	_, err = sh.Exec(
		composeEnv,
		os.Stdout,
		os.Stderr,
		"docker-compose",
		args...,
	)
	return err
}

func StopIntegTestContainers() error {
	// Docker-compose rm is noisy. So only pass through stderr when in verbose.
	out := ioutil.Discard
	if mg.Verbose() {
		out = os.Stderr
	}

	composeEnv, err := integTestDockerComposeEnvVars()
	if err != nil {
		return err
	}

	_, err = sh.Exec(
		composeEnv,
		ioutil.Discard,
		out,
		"docker-compose",
		"-p", DockerComposeProjectName(),
		"rm", "--stop", "--force",
	)

	return err
}

// DockerComposeProjectName returns the project name to use with docker-compose.
// It is passed to docker-compose using the `-p` flag. And is passed to our
// Go and Python testing libraries through the DOCKER_COMPOSE_PROJECT_NAME
// environment variable.
func DockerComposeProjectName() string {
	commit, err := CommitHash()
	if err != nil {
		panic(fmt.Errorf("failed to construct docker compose project name: %w", err))
	}

	version, err := BeatQualifiedVersion()
	if err != nil {
		panic(fmt.Errorf("failed to construct docker compose project name: %w", err))
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

func saveDockerComposeLogs(rootDir string, mageTarget string) error {
	var (
		composeLogDir      = filepath.Join(rootDir, "build", "system-tests", "docker-logs")
		composeLogFileName = filepath.Join(composeLogDir, "TEST-docker-compose-"+mageTarget+".log")
	)

	composeEnv, err := integTestDockerComposeEnvVars()
	if err != nil {
		return err
	}

	if err = os.MkdirAll(composeLogDir, os.ModeDir|os.ModePerm); err != nil {
		return fmt.Errorf("creating docker log dir: %w", err)
	}

	composeLogFile, err := os.Create(composeLogFileName)
	if err != nil {
		return fmt.Errorf("creating docker log file: %w", err)
	}
	defer composeLogFile.Close()

	_, err = sh.Exec(
		composeEnv,
		composeLogFile, // stdout
		composeLogFile, // stderr
		"docker-compose",
		"-p", DockerComposeProjectName(),
		"logs",
		"--no-color",
	)
	if err != nil {
		return fmt.Errorf("executing docker-compose logs: %w", err)
	}

	return nil
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

// WriteDockerComposeEnvFile generates a docker-compose environment variable file.
func WriteDockerComposeEnvFile() (string, error) {
	envFileContent := []string{
		"# Environment variable file to pass to docker-compose with the --env-file option.",
	}
	envVarMap, err := integTestDockerComposeEnvVars()
	if err != nil {
		return "", err
	}

	for k, v := range envVarMap {
		envFileContent = append(envFileContent, fmt.Sprintf("%s=%s", k, v))
	}

	esBeatsDir, err := ElasticBeatsDir()
	if err != nil {
		return "", err
	}

	envFile := filepath.Join(esBeatsDir, "docker.env")
	err = os.WriteFile(
		envFile,
		[]byte(strings.Join(envFileContent, "\n")),
		0644,
	)
	if err != nil {
		return "", err
	}

	return envFile, nil
}
