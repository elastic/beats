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

//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/elastic-agent-autodiscover/docker"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestHintsDocker(t *testing.T) {
	containerID := startFlogContainer(t)
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	cfgYAML := getConfig(t, nil, "autodiscover", "docker.yml")
	filebeat.WriteConfigFile(cfgYAML)
	filebeat.Start()

	// By ensuring the Filestream input started with the correct ID, we're
	// testing that the whole autodiscover + hints is working as expected.
	filebeat.WaitForLogs(
		fmt.Sprintf(
			`"message":"Input 'filestream' starting","service.name":"filebeat","id":"container-logs-%s"`,
			containerID,
		),
		30*time.Second,
		"Filestream did not start for the test container")
}

// startFlogContainer starts a `mingrammer/flog` that logs one line every
// second. The container ID is returned and the container is stopped at the
// end of the test. On error the test fails by calling t.Fatalf
func startFlogContainer(t *testing.T) string {
	ctx := t.Context()
	img := "mingrammer/flog"
	cli, err := docker.NewClient(client.DefaultDockerHost, nil, nil, logp.NewNopLogger())
	if err != nil {
		t.Fatalf("cannot create Docker client: %s", err)
	}

	resp, err := cli.ContainerCreate(
		ctx,
		&container.Config{
			Image: img,
			Cmd:   []string{"-l", "-d", "1", "-s", "1"},
		}, nil, nil, nil, "")
	if err != nil {
		t.Fatalf("cannot create container for %q: %s", img, err)
	}

	err = cli.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		t.Fatalf("cannot start container: %s", err)
	}

	t.Cleanup(func() {
		ctx := context.Background()
		if err := cli.ContainerStop(ctx, resp.ID, container.StopOptions{}); err != nil {
			t.Errorf("cannot stop container: %s", err)
		}
		if err := cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{}); err != nil {
			t.Errorf("cannot remove container: %s", err)
		}
	})
	return resp.ID
}
