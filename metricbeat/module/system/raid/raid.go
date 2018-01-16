package raid

import (
	"path/filepath"
	"regexp"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/system"
	"github.com/elastic/procfs"

	"github.com/pkg/errors"
)

func init() {
	if err := mb.Registry.AddMetricSet("system", "raid", New, parse.EmptyHostParser); err != nil {
		panic(err)
	}
}

// MetricSet contains proc fs data.
type MetricSet struct {
	mb.BaseMetricSet
	fs         procfs.FS
	nameRegexp *regexp.Regexp
}

// New creates a new instance of the raid metricset.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The system raid metricset is experimental")

	systemModule, ok := base.Module().(*system.Module)
	if !ok {
		return nil, errors.New("unexpected module type")
	}

	// Additional configuration options
	config := struct {
		MountPoint string `config:"raid.mount_point"`
		Regexp     string `config:"raid.name.regexp"`
	}{
		Regexp: "",
	}

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

	m := &MetricSet{
		BaseMetricSet: base,
		fs:            fs,
	}

	if len(config.Regexp) > 0 {
		r, err := regexp.Compile(config.Regexp)
		if err != nil {
			logp.Warn("raid", "Invalid regular expression: (%s), error: %s", config.Regexp, err.Error())
			return nil, err
		}
		m.nameRegexp = r
	}

	return m, nil
}

// Fetch fetches one event for each device
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	stats, err := m.fs.ParseMDStat()
	if err != nil {
		return nil, err
	}

	events := make([]common.MapStr, 0, len(stats))
	for _, stat := range stats {
		if m.nameRegexp != nil && !m.nameRegexp.MatchString(stat.Name) {
			continue
		}

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
