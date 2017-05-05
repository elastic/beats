package info

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/docker"

	dc "github.com/fsouza/go-dockerclient"
)

func init() {
	if err := mb.Registry.AddMetricSet("docker", "info", New, docker.HostParser); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	dockerClient *dc.Client
}

// New create a new instance of the docker info MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Beta("The docker info metricset is beta")

	config := docker.Config{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	client, err := docker.NewDockerClient(base.HostData().URI, config)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		dockerClient:  client,
	}, nil
}

// Fetch creates a new event for info.
// See: https://docs.docker.com/engine/reference/api/docker_remote_api_v1.24/#/display-system-wide-information
func (m *MetricSet) Fetch() (common.MapStr, error) {
	info, err := m.dockerClient.Info()
	if err != nil {
		return nil, err
	}

	return eventMapping(info), nil
}
