package diskio

import (
	dc "github.com/fsouza/go-dockerclient"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/docker"
)

func init() {
	if err := mb.Registry.AddMetricSet("docker", "diskio", New); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	blkioService *BLkioService
	dockerClient *dc.Client
}

// New create a new instance of the docker diskio MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	logp.Warn("EXPERIMENTAL: The diskio metricset is experimental")

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
		blkioService: &BLkioService{
			BlkioSTatsPerContainer: make(map[string]BlkioRaw),
		},
	}, nil
}

// Fetch creates list of events with diskio stats for all containers
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	stats, err := docker.FetchStats(m.dockerClient)
	if err != nil {
		return nil, err
	}

	formatedStats := m.blkioService.getBlkioStatsList(stats)
	return eventsMapping(formatedStats), nil
}
