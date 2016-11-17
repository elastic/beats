package container

import (
	dc "github.com/fsouza/go-dockerclient"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/elastic/beats/metricbeat/module/docker"
)

func init() {
	if err := mb.Registry.AddMetricSet("docker", "container", New); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	dockerClient *dc.Client
}

// New create a new instance of the container MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	logp.Warn("EXPERIMENTAL: The container metricset is experimental")

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
		dockerClient:  client,
	}, nil
}

// Fetch returns a list of all containers as events
// This is based on https://docs.docker.com/engine/reference/api/docker_remote_api_v1.24/#/list-containers
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	// Fetch list of all containers
	containers, err := m.dockerClient.ListContainers(dc.ListContainersOptions{})
	if err != nil {
		return nil, err
	}
	return eventsMapping(containers), nil
}
