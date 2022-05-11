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

//go:build (linux || darwin || windows) && !integration
// +build linux darwin windows
// +build !integration

package add_docker_metadata

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/bus"
	"github.com/elastic/beats/v7/libbeat/common/docker"
	"github.com/elastic/beats/v7/libbeat/metric/system/cgroup"
	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func init() {
	// Stub out the procfs.
	processCgroupPaths = func(_ resolve.Resolver, pid int) (cgroup.PathList, error) {

		switch pid {
		case 1000:
			return cgroup.PathList{
				V1: map[string]cgroup.ControllerPath{
					"cpu": {ControllerPath: "/docker/8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b", IsV2: false},
				},
			}, nil
		case 2000:
			return cgroup.PathList{
				V1: map[string]cgroup.ControllerPath{
					"memory": {ControllerPath: "/user.slice", IsV2: false},
				},
			}, nil
		case 3000:
			// Parser error (hopefully this never happens).
			return cgroup.PathList{}, fmt.Errorf("cgroup parse failure")
		default:
			return cgroup.PathList{}, os.ErrNotExist
		}
	}
}

func TestInitializationNoDocker(t *testing.T) {
	var testConfig = config.NewConfig()
	testConfig.SetString("host", -1, "unix:///var/run42/docker.sock")

	p, err := buildDockerMetadataProcessor(logp.L(), testConfig, docker.NewWatcher)
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := mapstr.M{}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	assert.Equal(t, mapstr.M{}, result.Fields)
}

func TestInitialization(t *testing.T) {
	var testConfig = config.NewConfig()

	p, err := buildDockerMetadataProcessor(logp.L(), testConfig, MockWatcherFactory(nil))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := mapstr.M{}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	assert.Equal(t, mapstr.M{}, result.Fields)
}

func TestNoMatch(t *testing.T) {
	testConfig, err := config.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"foo"},
	})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(logp.L(), testConfig, MockWatcherFactory(nil))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := mapstr.M{
		"field": "value",
	}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	assert.Equal(t, mapstr.M{"field": "value"}, result.Fields)
}

func TestMatchNoContainer(t *testing.T) {
	testConfig, err := config.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"foo"},
	})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(logp.L(), testConfig, MockWatcherFactory(nil))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := mapstr.M{
		"foo": "garbage",
	}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	assert.Equal(t, mapstr.M{"foo": "garbage"}, result.Fields)
}

func TestMatchContainer(t *testing.T) {
	testConfig, err := config.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"foo"},
		"labels.dedot": false,
	})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(logp.L(), testConfig, MockWatcherFactory(
		map[string]*docker.Container{
			"container_id": {
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

	input := mapstr.M{
		"foo": "container_id",
	}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	assert.EqualValues(t, mapstr.M{
		"container": mapstr.M{
			"id": "container_id",
			"image": mapstr.M{
				"name": "image",
			},
			"labels": mapstr.M{
				"a": mapstr.M{
					"x": "1",
				},
				"b": mapstr.M{
					"value": "2",
					"foo":   "3",
				},
			},
			"name": "name",
		},
		"foo": "container_id",
	}, result.Fields)
}

func TestMatchContainerWithDedot(t *testing.T) {
	testConfig, err := config.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"foo"},
	})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(logp.L(), testConfig, MockWatcherFactory(
		map[string]*docker.Container{
			"container_id": {
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

	input := mapstr.M{
		"foo": "container_id",
	}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	assert.EqualValues(t, mapstr.M{
		"container": mapstr.M{
			"id": "container_id",
			"image": mapstr.M{
				"name": "image",
			},
			"labels": mapstr.M{
				"a_x":   "1",
				"b":     "2",
				"b_foo": "3",
			},
			"name": "name",
		},
		"foo": "container_id",
	}, result.Fields)
}

func TestMatchSource(t *testing.T) {
	// Use defaults
	testConfig, err := config.NewConfigFrom(map[string]interface{}{})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(logp.L(), testConfig, MockWatcherFactory(
		map[string]*docker.Container{
			"8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b": {
				ID:    "8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b",
				Image: "image",
				Name:  "name",
				Labels: map[string]string{
					"a": "1",
					"b": "2",
				},
			},
		}))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	var inputSource string
	switch runtime.GOOS {
	case "windows":
		inputSource = "C:\\ProgramData\\docker\\containers\\FABADA\\foo.log"
	default:
		inputSource = "/var/lib/docker/containers/8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b/foo.log"
	}
	input := mapstr.M{
		"log": mapstr.M{
			"file": mapstr.M{
				"path": inputSource,
			},
		},
	}

	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	assert.EqualValues(t, mapstr.M{
		"container": mapstr.M{
			"id": "8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b",
			"image": mapstr.M{
				"name": "image",
			},
			"labels": mapstr.M{
				"a": "1",
				"b": "2",
			},
			"name": "name",
		},
		"log": mapstr.M{
			"file": mapstr.M{
				"path": inputSource,
			},
		},
	}, result.Fields)
}

func TestDisableSource(t *testing.T) {
	// Use defaults
	testConfig, err := config.NewConfigFrom(map[string]interface{}{
		"match_source": false,
	})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(logp.L(), testConfig, MockWatcherFactory(
		map[string]*docker.Container{
			"8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b": {
				ID:    "8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b",
				Image: "image",
				Name:  "name",
				Labels: map[string]string{
					"a": "1",
					"b": "2",
				},
			},
		}))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := mapstr.M{
		"source": "/var/lib/docker/containers/8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b/foo.log",
	}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	// remains unchanged
	assert.EqualValues(t, input, result.Fields)
}

func TestMatchPIDs(t *testing.T) {
	p, err := buildDockerMetadataProcessor(logp.L(), config.NewConfig(), MockWatcherFactory(
		map[string]*docker.Container{
			"8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b": {
				ID:    "8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b",
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

	dockerMetadata := mapstr.M{
		"container": mapstr.M{
			"id": "8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b",
			"image": mapstr.M{
				"name": "image",
			},
			"labels": mapstr.M{
				"a": "1",
				"b": "2",
			},
			"name": "name",
		},
	}

	t.Run("pid is not containerized", func(t *testing.T) {
		input := mapstr.M{}
		input.Put("process.pid", 2000)
		input.Put("process.parent.pid", 1000)

		expected := mapstr.M{}
		expected.DeepUpdate(input)

		result, err := p.Run(&beat.Event{Fields: input})
		assert.NoError(t, err, "processing an event")
		assert.EqualValues(t, expected, result.Fields)
	})

	t.Run("pid does not exist", func(t *testing.T) {
		input := mapstr.M{}
		input.Put("process.pid", 9999)

		expected := mapstr.M{}
		expected.DeepUpdate(input)

		result, err := p.Run(&beat.Event{Fields: input})
		assert.NoError(t, err, "processing an event")
		assert.EqualValues(t, expected, result.Fields)
	})

	t.Run("pid is containerized", func(t *testing.T) {
		fields := mapstr.M{}
		fields.Put("process.pid", "1000")

		expected := mapstr.M{}
		expected.DeepUpdate(dockerMetadata)
		expected.DeepUpdate(fields)

		result, err := p.Run(&beat.Event{Fields: fields})
		assert.NoError(t, err, "processing an event")
		assert.EqualValues(t, expected, result.Fields)
	})

	t.Run("pid exited and ppid is containerized", func(t *testing.T) {
		fields := mapstr.M{}
		fields.Put("process.pid", 9999)
		fields.Put("process.parent.pid", 1000)

		expected := mapstr.M{}
		expected.DeepUpdate(dockerMetadata)
		expected.DeepUpdate(fields)

		result, err := p.Run(&beat.Event{Fields: fields})
		assert.NoError(t, err, "processing an event")
		assert.EqualValues(t, expected, result.Fields)
	})

	t.Run("cgroup error", func(t *testing.T) {
		fields := mapstr.M{}
		fields.Put("process.pid", 3000)

		expected := mapstr.M{}
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
	return func(_ *logp.Logger, host string, tls *docker.TLSConfig, shortID bool) (docker.Watcher, error) {
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
