package cpu

import (
	dc "github.com/fsouza/go-dockerclient"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/docker"
)

func init() {
	if err := mb.Registry.AddMetricSet("docker", "cpu", New); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	cpuService   *CPUService
	dockerClient *dc.Client
}

// New create a new instance of the docker cpu MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	logp.Warn("EXPERIMENTAL: The cpu metricset is experimental")

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
		cpuService:    &CPUService{},
	}, nil
}

// Fetch returns a list of docker cpu stats
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	stats, err := docker.FetchStats(m.dockerClient)
	if err != nil {
		return nil, err
	}

	formatedStats := m.cpuService.getCPUStatsList(stats)
	return eventsMapping(formatedStats), nil

}
