package info

import (
	"context"

	"github.com/docker/docker/client"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/docker"
)

func init() {
	mb.Registry.MustAddMetricSet("docker", "info", New,
		mb.WithHostParser(docker.HostParser),
		mb.DefaultMetricSet(),
	)
}

type MetricSet struct {
	mb.BaseMetricSet
	dockerClient *client.Client
}

// New create a new instance of the docker info MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
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
	info, err := m.dockerClient.Info(context.TODO())
	if err != nil {
		return nil, err
	}

	return eventMapping(&info), nil
}
