package datastore

import (
	"context"
	"net/url"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
)

func init() {
	mb.Registry.MustAddMetricSet("vsphere", "datastore", New,
		mb.DefaultMetricSet(),
	)
}

type MetricSet struct {
	mb.BaseMetricSet
	HostURL  *url.URL
	Insecure bool
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The vsphere datastore metricset is beta")

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

	return &MetricSet{
		BaseMetricSet: base,
		HostURL:       u,
		Insecure:      config.Insecure,
	}, nil
}

func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var events []common.MapStr

	client, err := govmomi.NewClient(ctx, m.HostURL, m.Insecure)
	if err != nil {
		return nil, err
	}

	defer client.Logout(ctx)

	c := client.Client

	// Create a view of Datastore objects
	mgr := view.NewManager(c)

	v, err := mgr.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"Datastore"}, true)
	if err != nil {
		return nil, err
	}

	defer v.Destroy(ctx)

	// Retrieve summary property for all datastores
	var dst []mo.Datastore
	err = v.Retrieve(ctx, []string{"Datastore"}, []string{"summary"}, &dst)
	if err != nil {
		return nil, err
	}

	for _, ds := range dst {
		var usedSpacePercent int64
		if ds.Summary.Capacity > 0 {
			usedSpacePercent = 100 * (ds.Summary.Capacity - ds.Summary.FreeSpace) / ds.Summary.Capacity
		}
		usedSpaceBytes := ds.Summary.Capacity - ds.Summary.FreeSpace

		event := common.MapStr{
			"name":   ds.Summary.Name,
			"fstype": ds.Summary.Type,
			"capacity": common.MapStr{
				"total": common.MapStr{
					"bytes": ds.Summary.Capacity,
				},
				"free": common.MapStr{
					"bytes": ds.Summary.FreeSpace,
				},
				"used": common.MapStr{
					"bytes": usedSpaceBytes,
					"pct":   usedSpacePercent,
				},
			},
		}

		events = append(events, event)
	}

	return events, nil
}
