package host

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
	if err := mb.Registry.AddMetricSet("vsphere", "host", New); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	Client *vim25.Client
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Experimental("The vsphere host metricset is experimental")

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

	events := []common.MapStr{}
	for _, dc := range dcs {
		f.SetDatacenter(dc)

		hss, err := f.HostSystemList(ctx, "*")
		if err != nil {
			return nil, errors.Wrap(err, "failed to get hostsystem list")
		}

		pc := property.DefaultCollector(m.Client)

		// Convert hosts into list of references.
		var refs []types.ManagedObjectReference
		for _, hs := range hss {
			refs = append(refs, hs.Reference())
		}

		// Retrieve summary property (HostListSummary).
		var hst []mo.HostSystem
		err = pc.Retrieve(ctx, refs, []string{"summary"}, &hst)
		if err != nil {
			return nil, err
		}

		for _, hs := range hst {

			events = append(events, eventMapping(hs, dc.Name()))
		}
	}

	return events, nil
}
