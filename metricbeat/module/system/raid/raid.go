package raid

import (
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/prometheus/procfs"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/system"
)

func init() {
	mb.Registry.MustAddMetricSet("system", "raid", New,
		mb.WithHostParser(parse.EmptyHostParser),
	)
}

// MetricSet contains proc fs data.
type MetricSet struct {
	mb.BaseMetricSet
	fs procfs.FS
}

// New creates a new instance of the raid metricset.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The system raid metricset is beta")

	systemModule, ok := base.Module().(*system.Module)
	if !ok {
		return nil, errors.New("unexpected module type")
	}

	// Additional configuration options
	config := struct {
		MountPoint string `config:"raid.mount_point"`
	}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	if config.MountPoint == "" {
		config.MountPoint = systemModule.HostFS
	}

	mountPoint := filepath.Join(config.MountPoint, procfs.DefaultMountPoint)
	fs, err := procfs.NewFS(mountPoint)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		fs:            fs,
	}, nil
}

// Fetch fetches one event for each device
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	stats, err := m.fs.ParseMDStat()
	if err != nil {
		return nil, err
	}

	events := make([]common.MapStr, 0, len(stats))
	for _, stat := range stats {
		event := common.MapStr{
			"name":           stat.Name,
			"activity_state": stat.ActivityState,
			"disks": common.MapStr{
				"active": stat.DisksActive,
				"total":  stat.DisksTotal,
			},
			"blocks": common.MapStr{
				"synced": stat.BlocksSynced,
				"total":  stat.BlocksTotal,
			},
		}
		events = append(events, event)
	}

	return events, nil
}
