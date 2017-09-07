package add_docker_metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

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
		map[string]*Container{
			"container_id": &Container{
				ID:    "container_id",
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
					"a": "1",
					"b": "2",
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
		map[string]*Container{
			"FABADA": &Container{
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
		map[string]*Container{
			"FABADA": &Container{
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

// Mock container watcher

func MockWatcherFactory(containers map[string]*Container) WatcherConstructor {
	if containers == nil {
		containers = make(map[string]*Container)
	}
	return func(host string, tls *TLSConfig) (Watcher, error) {
		return &mockWatcher{containers: containers}, nil
	}
}

type mockWatcher struct {
	containers map[string]*Container
}

func (m *mockWatcher) Start() error {
	return nil
}

func (m *mockWatcher) Stop() {}

func (m *mockWatcher) Container(ID string) *Container {
	return m.containers[ID]
}

func (m *mockWatcher) Containers() map[string]*Container {
	return m.containers
}
