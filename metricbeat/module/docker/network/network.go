package network

import (
	dc "github.com/fsouza/go-dockerclient"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/docker"
)

func init() {
	if err := mb.Registry.AddMetricSet("docker", "network", New); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	netService   *NETService
	dockerClient *dc.Client
}

// New create a new instance of the docker network MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	logp.Warn("EXPERIMENTAL: The network metricset is experimental")

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
		netService: &NETService{
			NetworkStatPerContainer: make(map[string]map[string]NETRaw),
		},
	}, nil
}

// Fetch methods creates a list of network events for each container
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	stats, err := docker.FetchStats(m.dockerClient)
	if err != nil {
		return nil, err
	}

	formatedStats := m.netService.getNetworkStatsPerContainer(stats)
	return eventsMapping(formatedStats), nil
}
