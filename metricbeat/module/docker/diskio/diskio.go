package diskio

import (
	"github.com/docker/docker/client"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/docker"
)

func init() {
	if err := mb.Registry.AddMetricSet("docker", "diskio", New, docker.HostParser); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	blkioService *BLkioService
	dockerClient *client.Client
	dedot        bool
}

// New create a new instance of the docker diskio MetricSet.
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
		blkioService: &BLkioService{
			BlkioSTatsPerContainer: make(map[string]BlkioRaw),
		},
		dedot: config.DeDot,
	}, nil
}

// Fetch creates list of events with diskio stats for all containers.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	stats, err := docker.FetchStats(m.dockerClient, m.Module().Config().Timeout)
	if err != nil {
		return nil, err
	}

	formattedStats := m.blkioService.getBlkioStatsList(stats, m.dedot)
	return eventsMapping(formattedStats), nil
}
