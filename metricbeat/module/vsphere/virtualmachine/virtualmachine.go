package virtualmachine

import (
	"context"
	"net/url"
	"sync"

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
	if err := mb.Registry.AddMetricSet("vsphere", "virtualmachine", New); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	Client *vim25.Client
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Experimental("The vsphere virtualmachine metricset is experimental")

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

	var wg sync.WaitGroup
	var mutex sync.Mutex

	for _, dc := range dcs {
		f.SetDatacenter(dc)

		vms, err := f.VirtualMachineList(ctx, "*")
		if err != nil {
			return nil, errors.Wrap(err, "failed to get virtual machine list")
		}

		pc := property.DefaultCollector(m.Client)

		// Convert virtual machines into list of references.
		var refs []types.ManagedObjectReference
		for _, vm := range vms {
			refs = append(refs, vm.Reference())
		}

		// Retrieve summary property (VirtualMachineSummary).
		var vmt []mo.VirtualMachine
		err = pc.Retrieve(ctx, refs, []string{"summary"}, &vmt)
		if err != nil {
			return nil, err
		}

		for _, vm := range vmt {

			wg.Add(1)

			go func(vm mo.VirtualMachine) {

				defer wg.Done()

				freeMemory := (int64(vm.Summary.Config.MemorySizeMB) * 1024 * 1024) - (int64(vm.Summary.QuickStats.GuestMemoryUsage) * 1024 * 1024)

				event := common.MapStr{
					"datacenter": dc.Name(),
					"name":       vm.Summary.Config.Name,
					"cpu": common.MapStr{
						"used": common.MapStr{
							"mhz": vm.Summary.QuickStats.OverallCpuUsage,
						},
					},
					"memory": common.MapStr{
						"used": common.MapStr{
							"guest": common.MapStr{
								"bytes": vm.Summary.QuickStats.GuestMemoryUsage * 1024 * 1024,
							},
							"host": common.MapStr{
								"bytes": vm.Summary.QuickStats.HostMemoryUsage * 1024 * 1024,
							},
						},
						"total": common.MapStr{
							"guest": common.MapStr{
								"bytes": vm.Summary.Config.MemorySizeMB * 1024 * 1024,
							},
						},
						"free": common.MapStr{
							"guest": common.MapStr{
								"bytes": freeMemory,
							},
						},
					},
				}

				mutex.Lock()
				events = append(events, event)
				mutex.Unlock()
			}(vm)
		}
	}

	wg.Wait()

	return events, nil
}
