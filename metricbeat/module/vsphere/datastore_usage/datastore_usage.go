package datastore_usage

import (
	"context"
	"errors"
	"net/url"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func init() {
	if err := mb.Registry.AddMetricSet("vsphere", "datastore_usage", New); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	hostUrl  *url.URL
	insecure bool
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

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
		hostUrl:       u,
		insecure:      config.Insecure,
	}, nil
}

func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, err := govmomi.NewClient(ctx, m.hostUrl, m.insecure)
	if err != nil {
		return nil, err
	}

	f := find.NewFinder(c.Client, true)
	if f == nil {
		return nil, errors.New("Finder undefined for vsphere.")
	}

	// Get all datacenters
	dcs, err := f.DatacenterList(ctx, "*")
	if err != nil {
		return nil, err
	}

	events := []common.MapStr{}

	for _, dc := range dcs {

		f.SetDatacenter(dc)

		dss, err := f.DatastoreList(ctx, "*")
		if err != nil {
			return nil, err
		}

		pc := property.DefaultCollector(c.Client)

		// Convert datastores into list of references
		var refs []types.ManagedObjectReference
		for _, ds := range dss {
			refs = append(refs, ds.Reference())
		}

		// Retrieve summary property
		var dst []mo.Datastore
		err = pc.Retrieve(ctx, refs, []string{"summary"}, &dst)
		if err != nil {
			return nil, err
		}

		for _, ds := range dst {

			usedSpacePercent := 100 * (ds.Summary.Capacity - ds.Summary.FreeSpace) / ds.Summary.Capacity
			usedSpaceBytes := ds.Summary.Capacity - ds.Summary.FreeSpace

			event := common.MapStr{
				"datacenter": dc.Name(),
				"name":       ds.Summary.Name,
				"fstype":     ds.Summary.Type,
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
	}

	return events, nil
}
