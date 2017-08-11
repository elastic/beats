package host

import (
	"context"
	"net/url"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
)

func init() {
	if err := mb.Registry.AddMetricSet("vsphere", "host", New); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	Client *vim25.Client
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The vsphere host metricset is beta")

	config := struct {
		Username string `config:"username"`
		Password string `config:"password"`
		Insecure bool   `config:"insecure"`
	}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	u, err := url.Parse(base.HostData().URI)
	if err != nil {
		return nil, err
	}

	u.User = url.UserPassword(config.Username, config.Password)

	c, err := govmomi.NewClient(context.TODO(), u, config.Insecure)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		Client:        c.Client,
	}, nil
}

func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := []common.MapStr{}

	c := m.Client

	// Create a view of HostSystem objects.
	mgr := view.NewManager(c)

	v, err := mgr.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"HostSystem"}, true)
	if err != nil {
		return nil, err
	}

	defer v.Destroy(ctx)

	// Retrieve summary property for all hosts.
	var hst []mo.HostSystem
	err = v.Retrieve(ctx, []string{"HostSystem"}, []string{"summary"}, &hst)
	if err != nil {
		return nil, err
	}

	for _, hs := range hst {
		totalCpu := int64(hs.Summary.Hardware.CpuMhz) * int64(hs.Summary.Hardware.NumCpuCores)
		freeCpu := int64(totalCpu) - int64(hs.Summary.QuickStats.OverallCpuUsage)
		freeMemory := int64(hs.Summary.Hardware.MemorySize) - (int64(hs.Summary.QuickStats.OverallMemoryUsage) * 1024 * 1024)

		event := common.MapStr{
			"name": hs.Summary.Config.Name,
			"cpu": common.MapStr{
				"used": common.MapStr{
					"mhz": hs.Summary.QuickStats.OverallCpuUsage,
				},
				"total": common.MapStr{
					"mhz": totalCpu,
				},
				"free": common.MapStr{
					"mhz": freeCpu,
				},
			},
			"memory": common.MapStr{
				"used": common.MapStr{
					"bytes": (int64(hs.Summary.QuickStats.OverallMemoryUsage) * 1024 * 1024),
				},
				"total": common.MapStr{
					"bytes": hs.Summary.Hardware.MemorySize,
				},
				"free": common.MapStr{
					"bytes": freeMemory,
				},
			},
		}

		events = append(events, event)
	}

	return events, nil
}
