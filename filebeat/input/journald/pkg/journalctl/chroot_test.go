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

//go:build linux

package journalctl

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/testing/fs"
)

const (
	testTimeout     = 5 * time.Minute
	testImage       = "golang:latest"
	containerChroot = "/hostfs"
)

func TestNewFactoryChrootInDocker(t *testing.T) {
	if os.Getenv("IN_DOCKER_CONTAINER") != "true" {
		t.Skip("Skipping test - must run inside Docker container with IN_DOCKER_CONTAINER=true")
	}

	journalctlPath := os.Getenv("JOURNALCTL_PATH")
	if journalctlPath == "" {
		t.Fatal("environment variable JOURNALCTL_PATH not set")
	}

	// This test should be run inside the Docker container
	chrootPath := containerChroot

	// Check if journalctl exists in chroot
	fullPath := filepath.Join(chrootPath, journalctlPath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Skipf("journalctl not found at %s", fullPath)
	}

	testCtx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	tempDir := fs.TempDir(t)
	logger := logptest.NewFileLogger(t, tempDir)

	factory := NewFactory(chrootPath, journalctlPath)

	jctl, err := factory(testCtx, logger.Logger, "--version")
	require.NoError(t, err, "failed to create journalctl with chroot")
	defer jctl.Kill()

	// Try to read version output
	data, err := jctl.Next(testCtx)
	require.NoError(t, err, "failed to read from journalctl")
	require.NotEmpty(t, data, "expected output from journalctl --version")

	t.Logf("Successfully executed journalctl with chroot: %s", string(data))
}

func TestNewFactoryChroot(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), testTimeout)
	defer cancel()

	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err, "failed to create docker client")
	defer cli.Close()

	// Pull the image if not present
	pullImage(ctx, t, cli, testImage)

	// Get the absolute path to the current binary for mounting
	workDir, err := os.Getwd()
	require.NoError(t, err, "failed to get working directory")

	// Find the project root (where go.mod is)
	projectRoot := findProjectRoot(t, workDir)

	// TODO: fix this constant
	testDir := "/workspace/filebeat/input/journald/pkg/journalctl"

	// Find the absolute path to journalctl inside the chroot
	journalctlPath, err := exec.LookPath("journalctl")
	if err != nil {
		t.Fatalf("cannot look path for journalctl: %s", err)
	}

	t.Logf("Found journalctl at: %s", journalctlPath)

	// Create container configuration
	containerConfig := &container.Config{
		Image:      testImage,
		Cmd:        []string{"go", "test", "-v", "-count=1", "-run=TestNewFactoryChrootInDocker"},
		Tty:        true,
		WorkingDir: testDir,
		Env: []string{
			"IN_DOCKER_CONTAINER=true",
			fmt.Sprintf("JOURNALCTL_PATH=%s", journalctlPath),
		},
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   "/",
				Target:   containerChroot,
				ReadOnly: true,
			},
			{
				Type:   mount.TypeBind,
				Source: projectRoot,
				Target: "/workspace",
			},
		},
		Privileged: true, // Required for chroot
		AutoRemove: true,
	}

	// Create the container
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	require.NoError(t, err, "failed to create container")
	containerID := resp.ID

	// Start the container
	err = cli.ContainerStart(ctx, containerID, container.StartOptions{})
	require.NoError(t, err, "failed to start container")
	hr, err := cli.ContainerAttach(ctx, containerID, container.AttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		t.Fatalf("cannot attach to container: %d", err)
	}

	tempDir := fs.TempDir(t, "..", "..", "..", "..", "build")
	containerLogs := fs.NewLogFile(t, tempDir, "docker-container-*.log")
	go func() {
		_, err := io.Copy(containerLogs, hr.Reader)
		if err != nil {
			t.Logf("could not fully copy container logs: %s", err)
		}
	}()

	respChan, errchan := cli.ContainerWait(ctx, containerID, container.WaitConditionRemoved)
	select {
	case r := <-respChan:
		if r.StatusCode != 0 {
			t.Errorf("test in container returned status %d", r.StatusCode)
		}
	case err := <-errchan:
		t.Fatalf("error running container: %s", err)
	}
}

func pullImage(ctx context.Context, t *testing.T, cli *client.Client, imageName string) {
	reader, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	require.NoError(t, err, "failed to pull image")
	defer reader.Close()

	// Wait for pull to complete
	_, err = io.Copy(io.Discard, reader)
	require.NoError(t, err, "failed to read image pull output")
}

func findProjectRoot(t *testing.T, startDir string) string {
	dir := startDir
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding go.mod
			t.Logf("Warning: go.mod not found, using current directory: %s", startDir)
			return startDir
		}
		dir = parent
	}
}
