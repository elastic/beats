package diskio

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/docker"

	dc "github.com/fsouza/go-dockerclient"
)

func init() {
	if err := mb.Registry.AddMetricSet("docker", "diskio", New, docker.HostParser); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	blkioService *BLkioService
	dockerClient *dc.Client
}

// New create a new instance of the docker diskio MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Warn("BETA: The docker diskio metricset is beta")

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
		blkioService: &BLkioService{
			BlkioSTatsPerContainer: make(map[string]BlkioRaw),
		},
	}, nil
}

// Fetch creates list of events with diskio stats for all containers.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	stats, err := docker.FetchStats(m.dockerClient, m.Module().Config().Timeout)
	if err != nil {
		return nil, err
	}

	formattedStats := m.blkioService.getBlkioStatsList(stats)
	return eventsMapping(formattedStats), nil
}
