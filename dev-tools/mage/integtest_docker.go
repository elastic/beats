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
	"go/build"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var (
	// StackEnvironment specifies what testing environment
	// to use (like snapshot (default), latest, 5x). Formerly known as
	// TESTING_ENVIRONMENT.
	StackEnvironment = EnvOr("STACK_ENVIRONMENT", "snapshot")
)

func init() {
	RegisterIntegrationTester(&DockerIntegrationTester{})
}

type DockerIntegrationTester struct {
	buildImagesOnce sync.Once
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
	if err := HaveDocker(); err != nil {
		return err
	}
	if err := HaveDockerCompose(); err != nil {
		return err
	}
	return nil
}

// StepRequirements returns the steps required for this tester.
func (d *DockerIntegrationTester) StepRequirements() IntegrationTestSteps {
	return IntegrationTestSteps{&MageIntegrationTestStep{}}
}

// Test performs the tests with docker-compose.
func (d *DockerIntegrationTester) Test(dir string, mageTarget string, env map[string]string) error {
	var err error
	d.buildImagesOnce.Do(func() { err = dockerComposeBuildImages() })
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
	args := []string{"-p", dockerComposeProjectName(), "run",
		"-e", "DOCKER_COMPOSE_PROJECT_NAME=" + dockerComposeProjectName(),
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
		// Do not set ES_USER or ES_PATH in this file unless you intend to override
		// values set in all individual docker-compose files
		//		"-e", "ES_USER=admin",
		//		"-e", "ES_PASS=testing",
	}
	args, err = addUidGidEnvArgs(args)
	if err != nil {
		return err
	}
	for envVame, envVal := range env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", envVame, envVal))
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

	err = saveDockerComposeLogs(dir, mageTarget, composeEnv)
	if err != nil && testErr == nil {
		// saving docker-compose logs failed but the test didn't.
		return err
	}

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
	if err != nil && testErr == nil {
		// docker-compose rm failed but the test didn't
		return err
	}
	return testErr
}

func saveDockerComposeLogs(rootDir string, mageTarget string, composeEnv map[string]string) error {
	var (
		composeLogDir      = filepath.Join(rootDir, "build", "system-tests", "docker-logs")
		composeLogFileName = filepath.Join(composeLogDir, "TEST-docker-compose-"+mageTarget+".log")
	)

	if err := os.MkdirAll(composeLogDir, os.ModeDir|os.ModePerm); err != nil {
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
		"-p", dockerComposeProjectName(),
		"logs",
		"--no-color",
	)
	if err != nil {
		return fmt.Errorf("executing docker-compose logs: %w", err)
	}

	return nil
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

	args := []string{"-p", dockerComposeProjectName(), "build", "--force-rm"}
	if _, noCache := os.LookupEnv("DOCKER_NOCACHE"); noCache {
		args = append(args, "--no-cache")
	}

	if _, forcePull := os.LookupEnv("DOCKER_PULL"); forcePull {
		args = append(args, "--pull")
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

	// This sleep is to avoid hitting the docker build issues when resources are not available.
	if err != nil {
		fmt.Println(">> Building docker images again")
		time.Sleep(10)
		_, err = sh.Exec(
			composeEnv,
			out,
			os.Stderr,
			"docker-compose", args...,
		)
	}
	return err
}
