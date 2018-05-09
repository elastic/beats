package network

import (
	"github.com/docker/docker/client"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/docker"
)

func init() {
	mb.Registry.MustAddMetricSet("docker", "network", New,
		mb.WithHostParser(docker.HostParser),
		mb.DefaultMetricSet(),
	)
}

type MetricSet struct {
	mb.BaseMetricSet
	netService   *NetService
	dockerClient *client.Client
	dedot        bool
}

// New creates a new instance of the docker network MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := docker.DefaultConfig()
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
		dedot: config.DeDot,
	}, nil
}

// Fetch methods creates a list of network events for each container.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	stats, err := docker.FetchStats(m.dockerClient, m.Module().Config().Timeout)
	if err != nil {
		return nil, err
	}

	formattedStats := m.netService.getNetworkStatsPerContainer(stats, m.dedot)
	return eventsMapping(formattedStats), nil
}
