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

//go:build integration && linux

package integration

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/gofrs/uuid/v5"

	"github.com/elastic/elastic-agent-libs/testing/fs"
)

func TestJournaldChroot(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("Failed to create Docker client: %s", err)
	}
	defer cli.Close()

	imageName := "journald-chroot"
	syslogID := uuid.Must(uuid.NewV4()).String()

	generateJournaldLogs(t, syslogID, 5, 100)

	tempDir := fs.TempDir(t, filepath.Join("..", "..", "build", "integration-tests"))
	containerLogFile := fs.NewLogFile(t, tempDir, "container-logs-*.log")

	filebeatPath := buildFilebeatBinary(t, tempDir)
	buildDockerImage(t, cli, imageName, tempDir, filebeatPath)
	startDockerContainer(t, cli, imageName, syslogID, containerLogFile)
	assertJournalctlWorks(t, containerLogFile, syslogID)
}

func buildDockerImage(t *testing.T, cli *client.Client, imageName, tempDir, filebeatPath string) {
	buildContextDir := "testdata/journald_chroot"

	buildOptions := build.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: "Dockerfile", // Dockerfile path relative to the build context
	}
	buildContext := createDockerContext(t, buildContextDir, filebeatPath)
	resp, err := cli.ImageBuild(t.Context(), buildContext, buildOptions)
	if err != nil {
		t.Fatalf("Failed to build Docker image: %s", err)
	}
	defer resp.Body.Close()

	// Keep the logs in case something goes wrong
	f := fs.NewLogFile(t, tempDir, "docker-build-log-*.log")
	if _, err := io.Copy(f, resp.Body); err != nil {
		t.Logf("cannot read Docker build logs: %s", err)
	}
}

func buildFilebeatBinary(t *testing.T, tempDir string) string {
	filebeatPath := filepath.Join(tempDir, "filebeat_static")
	cmd := exec.Command(
		"go",
		"build",
		"-ldflags",
		"-extldflags \"-static\" -s",
		"-tags",
		"timetzdata",
		"-o",
		filebeatPath,
		"../../")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build Filebeat binary: %s\nOutput: %s", err, output)
	}

	return filebeatPath
}

func startDockerContainer(t *testing.T, cli *client.Client, imageName, syslogID string, logFile *fs.LogFile) string {
	ctx := t.Context()

	containerConfig := &container.Config{
		Image: imageName,
		Tty:   true,
		Env: []string{
			"SYSLOG_ID=" + syslogID,
		},
	}
	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: "/",
				Target: "/hostfs",
			},
		},
		CapAdd:     []string{"CAP_SYS_CHROOT"}, // Required for chroot
		AutoRemove: true,
	}

	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		t.Fatalf("Failed to create Docker container: %s", err)
	}

	// Attach to the container's logs
	attachResp, err := cli.ContainerAttach(ctx, resp.ID, container.AttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		t.Fatalf("Failed to attach to Docker container logs: %s", err)
	}

	// Stream logs to the log file
	go func() {
		defer attachResp.Close()
		if _, err := io.Copy(logFile, attachResp.Reader); err != nil {
			t.Logf("Error streaming container logs: %s", err)
		}
	}()

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		t.Fatalf("Failed to start Docker container: %s", err)
	}

	t.Cleanup(func() {
		// By the time t.Cleanup runs the test context is already cancelled
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := cli.ContainerStop(ctx, resp.ID, container.StopOptions{}); err != nil {
			t.Logf("Failed to stop container %s: %s\n", resp.ID, err)
		}
	})

	return resp.ID
}

func assertJournalctlWorks(t *testing.T, logFile *fs.LogFile, syslogID string) {
	t.Helper()
	// Wait for the log message "journalctl started"
	logFile.WaitLogsContains(t, "journalctl started", 30*time.Second, "journalctl did not start")
	for range 5 {
		logFile.WaitLogsContains(t, syslogID, 5*time.Second, "did not find event")
	}
}

func createDockerContext(t *testing.T, dir, filebeatPath string) io.Reader {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	// Add the directory contents to the tar archive
	err := filepath.Walk(dir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return fmt.Errorf("cannot get FileInfoHeader for %q: %w", file, err)
		}

		header.Name, _ = filepath.Rel(dir, file)
		if fi.IsDir() {
			header.Name += "/"
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !fi.IsDir() {
			f, err := os.Open(file)
			if err != nil {
				return fmt.Errorf("cannot open file: %w", err)
			}
			defer f.Close()

			if _, err := io.Copy(tw, f); err != nil {
				return fmt.Errorf("cannot read %q: %s", file, err)
			}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("cannot generate tar archive: %s", err)
	}

	// Add the Filebeat binary to the tar archive
	filebeatFile, err := os.Open(filebeatPath)
	if err != nil {
		t.Fatalf("failed to open Filebeat binary: %s", err)
	}
	defer filebeatFile.Close()

	fileInfo, err := filebeatFile.Stat()
	if err != nil {
		t.Fatalf("failed to stat Filebeat binary: %s", err)
	}

	header, err := tar.FileInfoHeader(fileInfo, filebeatPath)
	if err != nil {
		t.Fatalf("failed to create tar header for Filebeat binary: %s", err)
	}
	header.Name = "filebeat" // Place the binary in the root of the build context

	if err := tw.WriteHeader(header); err != nil {
		t.Fatalf("failed to write tar header for Filebeat binary: %s", err)
	}

	if _, err := io.Copy(tw, filebeatFile); err != nil {
		t.Fatalf("failed to copy Filebeat binary to tar archive: %s", err)
	}

	return buf
}
