package cpu

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/docker"

	dc "github.com/fsouza/go-dockerclient"
)

func init() {
	if err := mb.Registry.AddMetricSet("docker", "cpu", New, docker.HostParser); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	cpuService   *CPUService
	dockerClient *dc.Client
}

// New creates a new instance of the docker cpu MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Warn("EXPERIMENTAL: The docker cpu metricset is experimental")

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
		cpuService:    &CPUService{},
	}, nil
}

// Fetch returns a list of docker CPU stats.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	stats, err := docker.FetchStats(m.dockerClient)
	if err != nil {
		return nil, err
	}

	formattedStats := m.cpuService.getCPUStatsList(stats)
	return eventsMapping(formattedStats), nil
}
