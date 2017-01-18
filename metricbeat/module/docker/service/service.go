package service

import (
	dc "github.com/fsouza/go-dockerclient"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/docker"
)

func init() {
	if err := mb.Registry.AddMetricSet("docker", "service", New, docker.HostParser); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	dockerClient *dc.Client
}

// New creates a new instance of the docker services MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Warn("EXPERIMENTAL: The docker service metricset is experimental")

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

// Fetch returns a list of all services as events.
// This is based on https://docs.docker.com/engine/reference/api/docker_remote_api_v1.24/#/list-services.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	// Fetch a list of all services.
	services, err := m.dockerClient.ListServices(dc.ListServicesOptions{})
	if err != nil {
		return nil, err
	}

	return eventsMapping(services), nil
}
