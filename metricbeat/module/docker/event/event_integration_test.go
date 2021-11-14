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
// +build integration

package event

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	"github.com/elastic/beats/v7/auditbeat/core"
	"github.com/elastic/beats/v7/libbeat/common/docker"
	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	ms := mbtest.NewPushMetricSetV2WithContext(t, getConfig())
	var events []mb.Event
	done := make(chan interface{})
	go func() {
		events = mbtest.RunPushMetricSetV2WithContext(10*time.Second, 1, ms)
		close(done)
	}()

	createEvent(t)
	<-done

	if len(events) == 0 {
		t.Fatal("received no events")
	}
	assertNoErrors(t, events)

	beatEvent := mbtest.StandardizeEvent(ms, events[0], core.AddDatasetToEvent)
	mbtest.WriteEventToDataJSON(t, beatEvent, "")
}

func assertNoErrors(t *testing.T, events []mb.Event) {
	t.Helper()

	for _, e := range events {
		t.Log(e)

		if e.Error != nil {
			t.Errorf("received error: %+v", e.Error)
		}
	}
}

func createEvent(t *testing.T) {
	c, err := docker.NewClient(client.DefaultDockerHost, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	reader, err := c.ImagePull(context.Background(), "busybox", types.ImagePullOptions{})
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(os.Stdout, reader)
	reader.Close()

	resp, err := c.ContainerCreate(context.Background(), &container.Config{
		Image: "busybox",
		Cmd:   []string{"echo", "foo"},
	}, nil, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	c.ContainerRemove(context.Background(), resp.ID, types.ContainerRemoveOptions{})
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "docker",
		"metricsets": []string{"event"},
		"hosts":      []string{"unix:///var/run/docker.sock"},
	}
}
