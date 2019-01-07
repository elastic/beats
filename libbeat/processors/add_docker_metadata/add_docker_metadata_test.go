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

package add_docker_metadata

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/docker"
)

func init() {
	// Stub out the procfs.
	processCgroupPaths = func(_ string, pid int) (map[string]string, error) {
		switch pid {
		case 1000:
			return map[string]string{
				"cpu": "/docker/FABADA",
			}, nil
		case 2000:
			return map[string]string{
				"memory": "/user.slice",
			}, nil
		case 3000:
			// Parser error (hopefully this never happens).
			return nil, fmt.Errorf("cgroup parse failure")
		default:
			return nil, os.ErrNotExist
		}
	}
}

func TestInitialization(t *testing.T) {
	var testConfig = common.NewConfig()

	p, err := buildDockerMetadataProcessor(testConfig, MockWatcherFactory(nil))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := common.MapStr{}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	assert.Equal(t, common.MapStr{}, result.Fields)
}

func TestNoMatch(t *testing.T) {
	testConfig, err := common.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"foo"},
	})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(testConfig, MockWatcherFactory(nil))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := common.MapStr{
		"field": "value",
	}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	assert.Equal(t, common.MapStr{"field": "value"}, result.Fields)
}

func TestMatchNoContainer(t *testing.T) {
	testConfig, err := common.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"foo"},
	})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(testConfig, MockWatcherFactory(nil))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := common.MapStr{
		"foo": "garbage",
	}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	assert.Equal(t, common.MapStr{"foo": "garbage"}, result.Fields)
}

func TestMatchContainer(t *testing.T) {
	testConfig, err := common.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"foo"},
	})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(testConfig, MockWatcherFactory(
		map[string]*docker.Container{
			"container_id": &docker.Container{
				ID:    "container_id",
				Image: "image",
				Name:  "name",
				Labels: map[string]string{
					"a.x":   "1",
					"b":     "2",
					"b.foo": "3",
				},
			},
		}))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := common.MapStr{
		"foo": "container_id",
	}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	assert.EqualValues(t, common.MapStr{
		"docker": common.MapStr{
			"container": common.MapStr{
				"id":    "container_id",
				"image": "image",
				"labels": common.MapStr{
					"a": common.MapStr{
						"x": "1",
					},
					"b": common.MapStr{
						"value": "2",
						"foo":   "3",
					},
				},
				"name": "name",
			},
		},
		"foo": "container_id",
	}, result.Fields)
}

func TestMatchContainerWithDedot(t *testing.T) {
	testConfig, err := common.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"foo"},
		"labels.dedot": true,
	})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(testConfig, MockWatcherFactory(
		map[string]*docker.Container{
			"container_id": &docker.Container{
				ID:    "container_id",
				Image: "image",
				Name:  "name",
				Labels: map[string]string{
					"a.x":   "1",
					"b":     "2",
					"b.foo": "3",
				},
			},
		}))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := common.MapStr{
		"foo": "container_id",
	}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	assert.EqualValues(t, common.MapStr{
		"docker": common.MapStr{
			"container": common.MapStr{
				"id":    "container_id",
				"image": "image",
				"labels": common.MapStr{
					"a_x":   "1",
					"b":     "2",
					"b_foo": "3",
				},
				"name": "name",
			},
		},
		"foo": "container_id",
	}, result.Fields)
}

func TestMatchSource(t *testing.T) {
	// Use defaults
	testConfig, err := common.NewConfigFrom(map[string]interface{}{})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(testConfig, MockWatcherFactory(
		map[string]*docker.Container{
			"FABADA": &docker.Container{
				ID:    "FABADA",
				Image: "image",
				Name:  "name",
				Labels: map[string]string{
					"a": "1",
					"b": "2",
				},
			},
		}))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := common.MapStr{
		"source": "/var/lib/docker/containers/FABADA/foo.log",
	}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	assert.EqualValues(t, common.MapStr{
		"docker": common.MapStr{
			"container": common.MapStr{
				"id":    "FABADA",
				"image": "image",
				"labels": common.MapStr{
					"a": "1",
					"b": "2",
				},
				"name": "name",
			},
		},
		"source": "/var/lib/docker/containers/FABADA/foo.log",
	}, result.Fields)
}

func TestDisableSource(t *testing.T) {
	// Use defaults
	testConfig, err := common.NewConfigFrom(map[string]interface{}{
		"match_source": false,
	})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(testConfig, MockWatcherFactory(
		map[string]*docker.Container{
			"FABADA": &docker.Container{
				ID:    "FABADA",
				Image: "image",
				Name:  "name",
				Labels: map[string]string{
					"a": "1",
					"b": "2",
				},
			},
		}))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := common.MapStr{
		"source": "/var/lib/docker/containers/FABADA/foo.log",
	}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	// remains unchanged
	assert.EqualValues(t, input, result.Fields)
}

func TestMatchPIDs(t *testing.T) {
	p, err := buildDockerMetadataProcessor(common.NewConfig(), MockWatcherFactory(
		map[string]*docker.Container{
			"FABADA": &docker.Container{
				ID:    "FABADA",
				Image: "image",
				Name:  "name",
				Labels: map[string]string{
					"a": "1",
					"b": "2",
				},
			},
		},
	))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	dockerMetadata := common.MapStr{
		"docker": common.MapStr{
			"container": common.MapStr{
				"id":    "FABADA",
				"image": "image",
				"labels": common.MapStr{
					"a": "1",
					"b": "2",
				},
				"name": "name",
			},
		},
	}

	t.Run("pid is not containerized", func(t *testing.T) {
		input := common.MapStr{}
		input.Put("process.pid", 2000)
		input.Put("process.ppid", 1000)

		expected := common.MapStr{}
		expected.DeepUpdate(input)

		result, err := p.Run(&beat.Event{Fields: input})
		assert.NoError(t, err, "processing an event")
		assert.EqualValues(t, expected, result.Fields)
	})

	t.Run("pid does not exist", func(t *testing.T) {
		input := common.MapStr{}
		input.Put("process.pid", 9999)

		expected := common.MapStr{}
		expected.DeepUpdate(input)

		result, err := p.Run(&beat.Event{Fields: input})
		assert.NoError(t, err, "processing an event")
		assert.EqualValues(t, expected, result.Fields)
	})

	t.Run("pid is containerized", func(t *testing.T) {
		fields := common.MapStr{}
		fields.Put("process.pid", "1000")

		expected := common.MapStr{}
		expected.DeepUpdate(dockerMetadata)
		expected.DeepUpdate(fields)

		result, err := p.Run(&beat.Event{Fields: fields})
		assert.NoError(t, err, "processing an event")
		assert.EqualValues(t, expected, result.Fields)
	})

	t.Run("pid exited and ppid is containerized", func(t *testing.T) {
		fields := common.MapStr{}
		fields.Put("process.pid", 9999)
		fields.Put("process.ppid", 1000)

		expected := common.MapStr{}
		expected.DeepUpdate(dockerMetadata)
		expected.DeepUpdate(fields)

		result, err := p.Run(&beat.Event{Fields: fields})
		assert.NoError(t, err, "processing an event")
		assert.EqualValues(t, expected, result.Fields)
	})

	t.Run("cgroup error", func(t *testing.T) {
		fields := common.MapStr{}
		fields.Put("process.pid", 3000)

		expected := common.MapStr{}
		expected.DeepUpdate(fields)

		result, err := p.Run(&beat.Event{Fields: fields})
		assert.NoError(t, err, "processing an event")
		assert.EqualValues(t, expected, result.Fields)
	})
}

// Mock container watcher

func MockWatcherFactory(containers map[string]*docker.Container) docker.WatcherConstructor {
	if containers == nil {
		containers = make(map[string]*docker.Container)
	}
	return func(host string, tls *docker.TLSConfig, shortID bool) (docker.Watcher, error) {
		return &mockWatcher{containers: containers}, nil
	}
}

type mockWatcher struct {
	containers map[string]*docker.Container
}

func (m *mockWatcher) Start() error {
	return nil
}

func (m *mockWatcher) Stop() {}

func (m *mockWatcher) Container(ID string) *docker.Container {
	return m.containers[ID]
}

func (m *mockWatcher) Containers() map[string]*docker.Container {
	return m.containers
}

func (m *mockWatcher) ListenStart() bus.Listener {
	return nil
}

func (m *mockWatcher) ListenStop() bus.Listener {
	return nil
}
