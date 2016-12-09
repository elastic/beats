package network

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/docker"

	dc "github.com/fsouza/go-dockerclient"
)

func init() {
	if err := mb.Registry.AddMetricSet("docker", "network", New, docker.HostParser); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	netService   *NetService
	dockerClient *dc.Client
}

// New creates a new instance of the docker network MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Warn("EXPERIMENTAL: The docker network metricset is experimental")

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
		netService: &NetService{
			NetworkStatPerContainer: make(map[string]map[string]NetRaw),
		},
	}, nil
}

// Fetch methods creates a list of network events for each container.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	stats, err := docker.FetchStats(m.dockerClient)
	if err != nil {
		return nil, err
	}

	formattedStats := m.netService.getNetworkStatsPerContainer(stats)
	return eventsMapping(formattedStats), nil
}
