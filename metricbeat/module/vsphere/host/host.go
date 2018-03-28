package host

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func init() {
	mb.Registry.MustAddMetricSet("vsphere", "host", New,
		mb.DefaultMetricSet(),
	)
}

type MetricSet struct {
	mb.BaseMetricSet
	HostURL  *url.URL
	Insecure bool
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
		totalCPU := int64(hs.Summary.Hardware.CpuMhz) * int64(hs.Summary.Hardware.NumCpuCores)
		freeCPU := int64(totalCPU) - int64(hs.Summary.QuickStats.OverallCpuUsage)
		freeMemory := int64(hs.Summary.Hardware.MemorySize) - (int64(hs.Summary.QuickStats.OverallMemoryUsage) * 1024 * 1024)

		event := common.MapStr{
			"name": hs.Summary.Config.Name,
			"cpu": common.MapStr{
				"used": common.MapStr{
					"mhz": hs.Summary.QuickStats.OverallCpuUsage,
				},
				"total": common.MapStr{
					"mhz": totalCPU,
				},
				"free": common.MapStr{
					"mhz": freeCPU,
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

		if hs.Summary.Host != nil {
			networkNames, err := getNetworkNames(ctx, c, hs.Summary.Host.Reference())
			if err != nil {
				logp.Debug("vsphere", err.Error())
			} else {
				if len(networkNames) > 0 {
					event["network_names"] = networkNames
				}
			}
		}

		events = append(events, event)
	}

	return events, nil
}

func getNetworkNames(ctx context.Context, c *vim25.Client, ref types.ManagedObjectReference) ([]string, error) {
	var outputNetworkNames []string

	pc := property.DefaultCollector(c)

	var hs mo.HostSystem
	err := pc.RetrieveOne(ctx, ref, []string{"network"}, &hs)
	if err != nil {
		return nil, fmt.Errorf("error retrieving host information: %v", err)
	}

	if len(hs.Network) == 0 {
		return nil, errors.New("no networks found")
	}

	var networkRefs []types.ManagedObjectReference
	for _, obj := range hs.Network {
		if obj.Type == "Network" {
			networkRefs = append(networkRefs, obj)
		}
	}

	if len(networkRefs) == 0 {
		return nil, errors.New("no networks found")
	}

	var nets []mo.Network
	err = pc.Retrieve(ctx, networkRefs, []string{"name"}, &nets)
	if err != nil {
		return nil, fmt.Errorf("error retrieving network from host: %v", err)
	}

	for _, net := range nets {
		name := strings.Replace(net.Name, ".", "_", -1)
		outputNetworkNames = append(outputNetworkNames, name)
	}

	return outputNetworkNames, nil
}
