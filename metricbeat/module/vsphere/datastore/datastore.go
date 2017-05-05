package datastore

import (
	"context"
	"net/url"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/pkg/errors"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func init() {
	if err := mb.Registry.AddMetricSet("vsphere", "datastore", New); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	Client *vim25.Client
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Experimental("The vsphere datastore metricset is experimental")

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
	f := find.NewFinder(m.Client, true)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get all data centers.
	dcs, err := f.DatacenterList(ctx, "*")
	if err != nil {
		return nil, err
	}

	var events []common.MapStr
	for _, dc := range dcs {
		f.SetDatacenter(dc)

		dss, err := f.DatastoreList(ctx, "*")
		if err != nil {
			return nil, errors.Wrap(err, "failed to get datastore list")
		}

		pc := property.DefaultCollector(m.Client)

		// Convert datastores into list of references.
		var refs []types.ManagedObjectReference
		for _, ds := range dss {
			refs = append(refs, ds.Reference())
		}

		// Retrieve summary property.
		var dst []mo.Datastore
		err = pc.Retrieve(ctx, refs, []string{"summary"}, &dst)
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
