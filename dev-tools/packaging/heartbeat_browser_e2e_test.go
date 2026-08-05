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

package dev_tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	heartbeatBrowserE2EMonitorID = "generated-image-browser-e2e"
	heartbeatBrowserE2EConfig    = "testdata/heartbeat-browser-e2e.yml"
	heartbeatConfigPath          = "/usr/share/heartbeat/heartbeat.yml"
	heartbeatBrowserE2ETimeout   = 90 * time.Second
)

var heartbeatBrowserE2EArchivePattern = regexp.MustCompile(`^heartbeat-\d+\.\d+\.\d+(?:-[A-Za-z0-9.]+)*-linux-amd64\.docker\.tar\.gz$`)

// checkHeartbeatBrowserE2E runs a browser journey from the standard Linux AMD64
// Heartbeat image, if that image was generated as part of the package test.
func checkHeartbeatBrowserE2E(t *testing.T, dockerArchives []string) {
	t.Helper()

	t.Run("heartbeat browser journey", func(t *testing.T) {
		archive, err := heartbeatBrowserE2EArchive(dockerArchives)
		require.NoError(t, err, "selecting Heartbeat browser E2E archive should succeed")
		if archive == "" {
			t.Skip("standard Linux AMD64 Heartbeat Docker archive was not generated")
		}

		ctx, cancel := dockerTestContext(t)
		defer cancel()

		dockerClient, err := client.New(client.FromEnv)
		require.NoError(t, err, "creating Docker client should succeed")

		// Like the generic image-run check, this intentionally leaves the loaded image
		// in the daemon. This can accumulate images locally, but avoids cleanup races.
		imageID, err := loadDockerImageFromArchive(ctx, dockerClient, archive)
		require.NoError(t, err, "loading Docker image %q should succeed", archive)

		image, err := dockerClient.ImageInspect(ctx, imageID)
		require.NoError(t, err, "inspecting Docker image %q should succeed", imageID)
		require.Equal(t, "amd64", image.Architecture, "selected Docker image must be AMD64")
		require.Equal(t, "linux", image.Os, "selected Docker image must be Linux")

		configPath := copyHeartbeatBrowserE2EConfig(t)
		createResp, err := dockerClient.ContainerCreate(ctx, client.ContainerCreateOptions{
			Config: &container.Config{
				Image: imageID,
				Cmd:   []string{"--strict.perms=false"},
			},
			HostConfig: &container.HostConfig{
				Binds: []string{configPath + ":" + heartbeatConfigPath + ":ro"},
			},
		})
		require.NoError(t, err, "creating Heartbeat browser E2E container should succeed")
		defer func() {
			_, removeErr := dockerClient.ContainerRemove(context.Background(), createResp.ID, client.ContainerRemoveOptions{Force: true})
			if removeErr != nil {
				t.Logf("removing Heartbeat browser E2E container: %v", removeErr)
			}
		}()

		_, err = dockerClient.ContainerStart(ctx, createResp.ID, client.ContainerStartOptions{})
		require.NoError(t, err, "starting Heartbeat browser E2E container should succeed")

		waitCtx, waitCancel := context.WithTimeout(ctx, heartbeatBrowserE2ETimeout)
		defer waitCancel()
		wait := dockerClient.ContainerWait(waitCtx, createResp.ID, client.ContainerWaitOptions{Condition: container.WaitConditionNotRunning})

		var exitCode int64
		var waitErr error
		select {
		case response := <-wait.Result:
			exitCode = response.StatusCode
		case waitErr = <-wait.Error:
		case <-waitCtx.Done():
			waitErr = fmt.Errorf("container did not exit within %s: %w", heartbeatBrowserE2ETimeout, waitCtx.Err())
		}

		stdout, stderr, logsErr := heartbeatBrowserE2ELogs(ctx, dockerClient, createResp.ID)
		inspect, inspectErr := dockerClient.ContainerInspect(ctx, createResp.ID, client.ContainerInspectOptions{})
		inspectState := any("unavailable")
		if inspectErr == nil {
			inspectState = inspect.Container.State
		}
		diagnostics := fmt.Sprintf("stdout:\n%s\nstderr:\n%s\ninspect:\n%+v", stdout, stderr, inspectState)

		if waitErr != nil {
			t.Fatalf("waiting for Heartbeat browser E2E container failed: %v\n%s", waitErr, diagnostics)
		}
		require.NoError(t, logsErr, "reading Heartbeat browser E2E container logs should succeed")
		require.NoError(t, inspectErr, "inspecting Heartbeat browser E2E container should succeed")

		events := heartbeatBrowserE2EEvents(stdout)
		assert.Equal(t, int64(0), exitCode, "Heartbeat browser E2E container should exit successfully\n%s", diagnostics)
		assert.True(t, events.hasSuccessfulStep, "expected successful page loads step event\n%s", diagnostics)
		assert.True(t, events.hasSuccessfulSummary, "expected successful monitor summary event\n%s", diagnostics)
	})
}

func heartbeatBrowserE2EArchive(archives []string) (string, error) {
	var matches []string
	for _, archive := range archives {
		if isHeartbeatBrowserE2EArchive(filepath.Base(archive)) {
			matches = append(matches, archive)
		}
	}

	switch len(matches) {
	case 0:
		return "", nil
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("expected one standard Linux AMD64 Heartbeat Docker archive, found %d: %s", len(matches), strings.Join(matches, ", "))
	}
}

func isHeartbeatBrowserE2EArchive(name string) bool {
	return heartbeatBrowserE2EArchivePattern.MatchString(name)
}

func copyHeartbeatBrowserE2EConfig(t *testing.T) string {
	t.Helper()

	contents, err := os.ReadFile(heartbeatBrowserE2EConfig)
	require.NoError(t, err, "reading Heartbeat browser E2E fixture should succeed")

	path := filepath.Join(t.TempDir(), "heartbeat.yml")
	require.NoError(t, os.WriteFile(path, contents, 0o644), "writing Heartbeat browser E2E config should succeed")
	return path
}

type heartbeatBrowserE2EEventResult struct {
	hasSuccessfulStep    bool
	hasSuccessfulSummary bool
}

func heartbeatBrowserE2EEvents(stdout string) heartbeatBrowserE2EEventResult {
	var result heartbeatBrowserE2EEventResult
	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		var event map[string]any
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			continue
		}

		monitor, _ := event["monitor"].(map[string]any)
		synthetics, _ := event["synthetics"].(map[string]any)
		step, _ := synthetics["step"].(map[string]any)
		if synthetics["type"] == "step/end" && step["name"] == "page loads" && step["status"] == "succeeded" {
			result.hasSuccessfulStep = true
		}
		if monitor["id"] == heartbeatBrowserE2EMonitorID && monitor["status"] == "up" {
			if _, ok := event["summary"].(map[string]any); ok {
				result.hasSuccessfulSummary = true
			}
		}
	}
	return result
}

func heartbeatBrowserE2ELogs(ctx context.Context, dockerClient *client.Client, containerID string) (string, string, error) {
	logs, err := dockerClient.ContainerLogs(ctx, containerID, client.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", "", err
	}
	defer logs.Close()

	var stdout, stderr bytes.Buffer
	_, err = stdcopy.StdCopy(&stdout, &stderr, logs)
	if err != nil {
		return stdout.String(), stderr.String(), err
	}
	return stdout.String(), stderr.String(), nil
}

func TestHeartbeatBrowserE2EArchive(t *testing.T) {
	for _, test := range []struct {
		name     string
		archives []string
		want     string
		wantErr  bool
	}{
		{name: "standard archive", archives: []string{"heartbeat-9.5.0-SNAPSHOT-linux-amd64.docker.tar.gz"}, want: "heartbeat-9.5.0-SNAPSHOT-linux-amd64.docker.tar.gz"},
		{name: "no matching archive", archives: []string{"heartbeat-wolfi-9.5.0-linux-amd64.docker.tar.gz"}},
		{name: "unexpected variant", archives: []string{"heartbeat-custom-9.5.0-linux-amd64.docker.tar.gz"}},
		{name: "excluded variants", archives: []string{"heartbeat-oss-9.5.0-linux-amd64.docker.tar.gz", "heartbeat-ubi-9.5.0-linux-amd64.docker.tar.gz", "heartbeat-wolfi-9.5.0-linux-amd64.docker.tar.gz", "heartbeat-fips-9.5.0-linux-amd64.docker.tar.gz", "heartbeat-9.5.0-linux-arm64.docker.tar.gz"}},
		{name: "ambiguous archives", archives: []string{"heartbeat-9.4.0-linux-amd64.docker.tar.gz", "heartbeat-9.5.0-linux-amd64.docker.tar.gz"}, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := heartbeatBrowserE2EArchive(test.archives)
			if test.wantErr {
				require.Error(t, err, "selecting an ambiguous archive should fail")
				return
			}
			require.NoError(t, err, "selecting archive should not fail")
			assert.Equal(t, test.want, got, "selected archive should match")
		})
	}
}

func TestHeartbeatBrowserE2EEvents(t *testing.T) {
	stdout := strings.Join([]string{
		"unrelated startup log",
		"{not-json}",
		`{"synthetics":{"type":"step/end","step":{"name":"other step","status":"succeeded"}}}`,
		`{"synthetics":{"type":"step/end","step":{"name":"page loads","status":"succeeded"}}}`,
		`{"monitor":{"id":"generated-image-browser-e2e","status":"up"},"summary":{}}`,
	}, "\n")

	result := heartbeatBrowserE2EEvents(stdout)
	assert.True(t, result.hasSuccessfulStep, "successful step should be found")
	assert.True(t, result.hasSuccessfulSummary, "successful summary should be found")
}

func TestHeartbeatBrowserE2EEventsMissingRequiredEvents(t *testing.T) {
	result := heartbeatBrowserE2EEvents(`{"synthetics":{"type":"step/end","step":{"name":"page loads","status":"failed"}}}`)
	assert.False(t, result.hasSuccessfulStep, "failed step must not satisfy assertion")
	assert.False(t, result.hasSuccessfulSummary, "missing summary must not satisfy assertion")
}
