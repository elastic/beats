package add_docker_metadata

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
	"github.com/elastic/beats/libbeat/processors/add_docker_metadata"
)

func TestLogPath(t *testing.T) {
	testConfig, err := common.NewConfigFrom(map[string]interface{}{
		"logs_path": "/path1/path2/",
	})
	assert.NoError(t, err)

	p, err := add_docker_metadata.BuildDockerMetadataProcessor(*testConfig, MockWatcherFactory(
		map[string]*add_docker_metadata.Container{
			"9b11fc6df837c05fe81a174b80fb3731c32a5dba442af6146944cb0f85e30e56": &add_docker_metadata.Container{
				ID:    "9b11fc6df837c05fe81a174b80fb3731c32a5dba442af6146944cb0f85e30e56",
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
		"source": "/path1/path2/9b11fc6df837c05fe81a174b80fb3731c32a5dba442af6146944cb0f85e30e56/container.log",
	}
	result, err := p.Run(input)
	assert.NoError(t, err, "processing an event")

	assert.EqualValues(t, common.MapStr{
		"source":"/path1/path2/9b11fc6df837c05fe81a174b80fb3731c32a5dba442af6146944cb0f85e30e56/container.log",
		"docker": common.MapStr{
			"container": common.MapStr{
				"id":    "9b11fc6df837c05fe81a174b80fb3731c32a5dba442af6146944cb0f85e30e56",
				"image": "image",
				"labels": common.MapStr{
					"a": "1",
					"b": "2",
				},
				"name": "name",
			},
		},
	}, result)
}

func MockWatcherFactory(containers map[string]*add_docker_metadata.Container) add_docker_metadata.WatcherConstructor {
	if containers == nil {
		containers = make(map[string]*add_docker_metadata.Container)
	}
	return func(host string, tls *add_docker_metadata.TLSConfig) (add_docker_metadata.Watcher, error) {
		return &mockWatcher{containers: containers}, nil
	}
}

type mockWatcher struct {
	containers map[string]*add_docker_metadata.Container
}

func (m *mockWatcher) Start() error {
	return nil
}

func (m *mockWatcher) Container(ID string) *add_docker_metadata.Container {
	return m.containers[ID]
}

func (m *mockWatcher) Containers() map[string]*add_docker_metadata.Container {
	return m.containers
}




