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

// This file was contributed to by generative AI

//go:build linux && integration

package journalctl

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/testing/fs"
)

// TestNewFactoryChroot starts a docker container mounting / as /hostfs and
// the current directory as /workspace. The container runs TestInDockerNewFactory
// that sets the chroot to access the host's journalctl
func TestNewFactoryChroot(t *testing.T) {
	containerChroot := "/hostfs"

	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err, "failed to create docker client")
	defer cli.Close()

	// Find the project root (where go.mod is)
	projectRoot := findProjectRoot(t)
	imageName := getImageName(t, projectRoot)

	// Pull the image if not present
	pullImage(t, cli, imageName)

	// Find the absolute path to journalctl inside the chroot
	journalctlPath, err := exec.LookPath("journalctl")
	require.NoError(t, err, "cannot look path for journalctl")

	tempDir := fs.TempDir(t, "..", "..", "..", "..", "build", "integration-tests")

	// Create container configuration
	containerConfig := &container.Config{
		Image:       imageName,
		Cmd:         []string{"go", "test", "-v", "-count=1", "-tags=integration", "-run=TestInDockerNewFactory"},
		Tty:         true,
		AttachStdin: false,
		WorkingDir:  "/workspace/filebeat/input/journald/pkg/journalctl",
		Env: []string{
			"IN_DOCKER_CONTAINER=true",
			fmt.Sprintf("JOURNALCTL_PATH=%s", journalctlPath),
			fmt.Sprintf("CHROOT_PATH=%s", containerChroot),
			fmt.Sprintf("TEST_TEMP_DIR=%s", filepath.Join(containerChroot, tempDir)),
		},
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: "/",
				Target: containerChroot,
			},
			{
				Type:   mount.TypeBind,
				Source: projectRoot,
				Target: "/workspace",
			},
		},
		CapAdd:     []string{"CAP_SYS_CHROOT"}, // Required for chroot
		AutoRemove: true,
	}

	// Create the container
	ctx := t.Context()
	createResp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	require.NoError(t, err, "failed to create container")
	containerID := createResp.ID

	// Start the container
	err = cli.ContainerStart(ctx, containerID, container.StartOptions{})
	require.NoError(t, err, "failed to start container")
	attachResp, err := cli.ContainerAttach(ctx, containerID, container.AttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	require.NoErrorf(t, err, "cannot attach to container: %d", err)

	containerLogs := fs.NewLogFile(t, tempDir, "docker-container-*.log")
	go func() {
		_, err := io.Copy(containerLogs, attachResp.Reader)
		if err != nil {
			t.Logf("could not fully copy container logs: %s", err)
		}
	}()

	waitRespChan, waitErrChan := cli.ContainerWait(ctx, containerID, container.WaitConditionRemoved)
	select {
	case r := <-waitRespChan:
		if r.StatusCode != 0 {
			t.Errorf("Test in container failed, returned status: %d.", r.StatusCode)
			if r.Error != nil {
				t.Logf("ContainerWait response error: %s", r.Error.Message)
			}

			logDockerCmd(t, imageName, containerConfig, hostConfig)()
			t.Log("Check the docker container logs for more information")
		}
	case err := <-waitErrChan:
		t.Fatalf("error waiting for container to finish: %s", err)
	}
}

func TestInDockerNewFactory(t *testing.T) {
	if os.Getenv("IN_DOCKER_CONTAINER") != "true" {
		t.Skip("Skipping test - must run inside Docker container with IN_DOCKER_CONTAINER=true")
	}

	journalctlPath := os.Getenv("JOURNALCTL_PATH")
	require.NotEmpty(t, journalctlPath, "JOURNALCTL_PATH must be set")

	chrootPath := os.Getenv("CHROOT_PATH")
	require.NotEmpty(t, chrootPath, "CHROOT_PATH must be set")

	tempDir := os.Getenv("TEST_TEMP_DIR")
	require.NotEmpty(t, tempDir, "TEST_TEMP_DIR be set")

	jctlCtx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	logger := logptest.NewFileLogger(t, tempDir)
	factory := NewFactory(chrootPath, journalctlPath)

	// Try to read version output, this ensures we can call journalctl
	// without the need of any messages in the journal
	jctl, err := factory(jctlCtx, logger.Logger, "--version")
	require.NoError(t, err, "failed to create journalctl with chroot")
	defer jctl.Kill() // nolint: deadcode // It's a test, there is nothing to do

	data, err := jctl.Next(jctlCtx)
	require.NoError(t, err, "failed to read from journalctl")
	require.NotEmpty(t, data, "expected output from journalctl --version")
}

func pullImage(t *testing.T, cli *client.Client, imageName string) {
	reader, err := cli.ImagePull(t.Context(), imageName, image.PullOptions{})
	require.NoErrorf(t, err, "failed to pull image: %q", imageName)
	defer reader.Close()

	// Wait for pull to complete
	_, err = io.Copy(io.Discard, reader)
	require.NoError(t, err, "failed to read image pull output")
}

func findProjectRoot(t *testing.T) string {
	startDir, err := os.Getwd()
	require.NoError(t, err, "failed to get working directory")

	// Add a level so we start looking at the current directory
	dir := filepath.Join(startDir, "foo")
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == "/" {
			// Reached root without finding go.mod
			t.Fatal("go.mod not found")
		}

		dir = parent
	}
}

func getImageName(t *testing.T, projectRoot string) string {
	// Construct the path to the .go-version file
	goVersionPath := filepath.Join(projectRoot, ".go-version")

	// Read the contents of the .go-version file
	data, err := os.ReadFile(goVersionPath)
	require.NoError(t, err, "failed to read .go-version file")

	// Trim leading and trailing spaces from the version string
	version := strings.TrimSpace(string(data))

	imageName := "golang:" + version + "-alpine"
	return imageName
}

func logDockerCmd(
	t *testing.T,
	imageName string,
	containerConfig *container.Config,
	hostConfig *container.HostConfig) func() {
	return func() {
		t.Logf("To reproduce, you can run the following Docker command:")

		// Construct the environment variables
		var envVars []string
		for _, env := range containerConfig.Env {
			envVars = append(envVars, "-e "+env)
		}

		// Construct the volume mounts
		var volumeMounts []string
		for _, m := range hostConfig.Mounts {
			mountOption := "-v " + m.Source + ":" + m.Target
			volumeMounts = append(volumeMounts, mountOption)
		}

		// Construct the docker run command
		dockerRunCmd := strings.Join([]string{
			"docker run --rm",
			strings.Join(envVars, " "),
			strings.Join(volumeMounts, " "),
			"-w " + containerConfig.WorkingDir,
			imageName,
			strings.Join(containerConfig.Cmd, " "),
		}, " ")

		t.Log(dockerRunCmd)
	}
}
