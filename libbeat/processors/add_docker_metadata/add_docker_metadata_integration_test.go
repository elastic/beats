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

// +build linux darwin windows
// +build integration

package add_docker_metadata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/docker"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	dockertest "github.com/elastic/beats/libbeat/tests/docker"
	"github.com/elastic/beats/libbeat/tests/resources"
)

func TestAddDockerMetadata(t *testing.T) {
	// Start a container to have some data to enrich an event
	testClient, err := dockertest.NewClient()
	require.NoError(t, err)
	image := "busybox"
	cmd := []string{"sleep", "60"}
	labels := map[string]string{"label": "foo"}
	id, err := testClient.ContainerStart(image, cmd, labels)
	require.NoError(t, err)
	defer testClient.ContainerKill(id)

	info, err := testClient.ContainerInspect(id)
	require.NoError(t, err)
	pid := info.State.Pid

	// Run the test under the goroutine leak checker
	resources.CallAndCheckGoroutines(t, func() {
		// Explicitly close the docker client, what only closes the idle keep-alive connections.
		// These idle connections affect the goroutines checker and will be eventually
		// closed in any case, so they don't represent a leak in principle.
		client, err := docker.NewClient(defaultConfig().Host, nil, nil)
		require.NoError(t, err)
		defer client.Close()

		config, err := common.NewConfigFrom(map[string]interface{}{
			"match_fields": []string{"cid"},
		})
		watcherConstructor := newWatcherWith(client)
		processor, err := buildDockerMetadataProcessor(logp.L(), config, watcherConstructor)
		require.NoError(t, err)

		t.Run("match container by container id", func(t *testing.T) {
			input := &beat.Event{Fields: common.MapStr{
				"cid": id,
			}}
			result, err := processor.Run(input)
			require.NoError(t, err)

			resultLabels, _ := result.Fields.GetValue("container.labels")
			expectedLabels := common.MapStr{"label": "foo"}
			assert.Equal(t, expectedLabels, resultLabels)
			assert.Equal(t, id, result.Fields["cid"])
		})

		t.Run("match container by process id", func(t *testing.T) {
			input := &beat.Event{Fields: common.MapStr{
				"cid":         id,
				"process.pid": pid,
			}}
			result, err := processor.Run(input)
			require.NoError(t, err)

			resultLabels, _ := result.Fields.GetValue("container.labels")
			expectedLabels := common.MapStr{"label": "foo"}
			assert.Equal(t, expectedLabels, resultLabels)
			assert.Equal(t, id, result.Fields["cid"])
		})

		t.Run("don't enrich non existing container", func(t *testing.T) {
			input := &beat.Event{Fields: common.MapStr{
				"cid": "notexists",
			}}
			result, err := processor.Run(input)
			require.NoError(t, err)
			assert.Equal(t, input.Fields, result.Fields)
		})

		err = processor.(processors.Closer).Close()
		require.NoError(t, err)
	})
}

func newWatcherWith(client docker.Client) docker.WatcherConstructor {
	return func(log *logp.Logger, host string, tls *docker.TLSConfig, storeShortID bool) (docker.Watcher, error) {
		return docker.NewWatcherWithClient(log, client, 60*time.Second, storeShortID)
	}
}
