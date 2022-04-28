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

//go:build (linux || darwin || windows) && integration
// +build linux darwin windows
// +build integration

package add_docker_metadata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/docker"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
	dockertest "github.com/elastic/beats/v7/libbeat/tests/docker"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestAddDockerMetadata(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	client, err := docker.NewClient(defaultConfig().Host, nil, nil)
	require.NoError(t, err)

	// Docker clients can affect the goroutines checker because they keep
	// idle keep-alive connections, so we explicitly close them.
	// These idle connections in principle wouldn't represent leaks even if
	// the client is not explicitly closed because they are eventually closed.
	defer client.Close()

	// Start a container to have some data to enrich events
	testClient, err := dockertest.NewClient()
	require.NoError(t, err)
	// Explicitly close client to don't affect goroutines checker
	defer testClient.Close()

	image := "busybox"
	cmd := []string{"sleep", "60"}
	labels := map[string]string{"label": "foo"}
	id, err := testClient.ContainerStart(image, cmd, labels)
	require.NoError(t, err)
	defer testClient.ContainerRemove(id)

	info, err := testClient.ContainerInspect(id)
	require.NoError(t, err)
	pid := info.State.Pid

	config, err := common.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"cid"},
	})
	watcherConstructor := newWatcherWith(client)
	processor, err := buildDockerMetadataProcessor(logp.L(), config, watcherConstructor)
	require.NoError(t, err)

	t.Run("match container by container id", func(t *testing.T) {
		input := &beat.Event{Fields: mapstr.M{
			"cid": id,
		}}
		result, err := processor.Run(input)
		require.NoError(t, err)

		resultLabels, _ := result.Fields.GetValue("container.labels")
		expectedLabels := mapstr.M{"label": "foo"}
		assert.Equal(t, expectedLabels, resultLabels)
		assert.Equal(t, id, result.Fields["cid"])
	})

	t.Run("match container by process id", func(t *testing.T) {
		input := &beat.Event{Fields: mapstr.M{
			"cid":         id,
			"process.pid": pid,
		}}
		result, err := processor.Run(input)
		require.NoError(t, err)

		resultLabels, _ := result.Fields.GetValue("container.labels")
		expectedLabels := mapstr.M{"label": "foo"}
		assert.Equal(t, expectedLabels, resultLabels)
		assert.Equal(t, id, result.Fields["cid"])
	})

	t.Run("don't enrich non existing container", func(t *testing.T) {
		input := &beat.Event{Fields: mapstr.M{
			"cid": "notexists",
		}}
		result, err := processor.Run(input)
		require.NoError(t, err)
		assert.Equal(t, input.Fields, result.Fields)
	})

	err = processors.Close(processor)
	require.NoError(t, err)
}

func newWatcherWith(client docker.Client) docker.WatcherConstructor {
	return func(log *logp.Logger, host string, tls *docker.TLSConfig, storeShortID bool) (docker.Watcher, error) {
		return docker.NewWatcherWithClient(log, client, 60*time.Second, storeShortID)
	}
}
