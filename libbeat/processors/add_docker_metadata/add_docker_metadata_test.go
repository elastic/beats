package add_docker_metadata

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestInitialization(t *testing.T) {
	var testConfig = common.NewConfig()

	p, err := buildDockerMetadataProcessor(*testConfig, MockWatcherFactory(nil))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := common.MapStr{}
	result, err := p.Run(input)
	assert.NoError(t, err, "processing an event")

	assert.Equal(t, common.MapStr{}, result)
}

func TestNoMatch(t *testing.T) {
	testConfig, err := common.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"foo"},
	})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(*testConfig, MockWatcherFactory(nil))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := common.MapStr{
		"field": "value",
	}
	result, err := p.Run(input)
	assert.NoError(t, err, "processing an event")

	assert.Equal(t, common.MapStr{"field": "value"}, result)
}

func TestMatchNoContainer(t *testing.T) {
	testConfig, err := common.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"foo"},
	})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(*testConfig, MockWatcherFactory(nil))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := common.MapStr{
		"foo": "garbage",
	}
	result, err := p.Run(input)
	assert.NoError(t, err, "processing an event")

	assert.Equal(t, common.MapStr{"foo": "garbage"}, result)
}

func TestMatchContainer(t *testing.T) {
	testConfig, err := common.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"foo"},
	})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(*testConfig, MockWatcherFactory(
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
	result, err := p.Run(input)
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
	}, result)
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

func (m *mockWatcher) Container(ID string) *Container {
	return m.containers[ID]
}

func (m *mockWatcher) Containers() map[string]*Container {
	return m.containers
}
