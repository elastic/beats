package memory

import (
	dc "github.com/fsouza/go-dockerclient"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/docker"
)

func init() {
	if err := mb.Registry.AddMetricSet("docker", "memory", New); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	memoryService *MemoryService
	dockerClient  *dc.Client
}

// New create a new instance of the  docker memory MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	logp.Warn("EXPERIMENTAL: The memory metricset is experimental")

	config := docker.GetDefaultConf()

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	client, err := docker.NewDockerClient(&config)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		memoryService: &MemoryService{},
		dockerClient:  client,
	}, nil
}

// Fetch creates a list of memory events for each container
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	stats, err := docker.FetchStats(m.dockerClient)
	if err != nil {
		return nil, err
	}

	memoryStats := m.memoryService.getMemoryStatsList(stats)
	return eventsMapping(memoryStats), nil
}
