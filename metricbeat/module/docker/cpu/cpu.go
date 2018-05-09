package cpu

import (
	"github.com/docker/docker/client"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/docker"
)

func init() {
	mb.Registry.MustAddMetricSet("docker", "cpu", New,
		mb.WithHostParser(docker.HostParser),
		mb.DefaultMetricSet(),
	)
}

type MetricSet struct {
	mb.BaseMetricSet
	cpuService   *CPUService
	dockerClient *client.Client
	dedot        bool
}

// New creates a new instance of the docker cpu MetricSet.
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
		cpuService:    &CPUService{},
		dedot:         config.DeDot,
	}, nil
}

// Fetch returns a list of docker CPU stats.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	stats, err := docker.FetchStats(m.dockerClient, m.Module().Config().Timeout)
	if err != nil {
		return nil, err
	}

	formattedStats := m.cpuService.getCPUStatsList(stats, m.dedot)
	return eventsMapping(formattedStats), nil
}
