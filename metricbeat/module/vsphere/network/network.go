package network

import (
	"context"
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/vsphere"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each network is defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("vsphere", "network", New,
		mb.WithHostParser(vsphere.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet.
type MetricSet struct {
	*vsphere.MetricSet
}

// New creates a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := vsphere.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{ms}, nil
}

type metricData struct {
	assetsName assetNames
}

type assetNames struct {
	outputVmNames   []string
	outputHostNames []string
}

// Fetch method implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) error {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	client, err := govmomi.NewClient(ctx, m.HostURL, m.Insecure)
	if err != nil {
		return fmt.Errorf("error in NewClient: %w", err)
	}

	defer func() {
		if err := client.Logout(ctx); err != nil {
			m.Logger().Errorf("error trying to log out from vSphere: %w", err)
		}
	}()

	c := client.Client

	// Create a view of Network objects.
	mgr := view.NewManager(c)

	v, err := mgr.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"Network"}, true)
	if err != nil {
		return fmt.Errorf("error in CreateContainerView: %w", err)
	}

	defer func() {
		if err := v.Destroy(ctx); err != nil {
			m.Logger().Errorf("error trying to destroy view from vSphere: %w", err)
		}
	}()

	// Retrieve property for all networks
	var nets []mo.Network
	err = v.Retrieve(ctx, []string{"Network"}, []string{"summary", "name", "overallStatus", "configStatus", "vm", "host", "name"}, &nets)
	if err != nil {
		return fmt.Errorf("error in Retrieve: %w", err)
	}

	pc := property.DefaultCollector(c)
	for i := range nets {
		assetNames, err := getAssetNames(ctx, pc, &nets[i])
		if err != nil {
			m.Logger().Errorf("Failed to retrieve object from network %s: %w", nets[i].Name, err)
		}

		reporter.Event(mb.Event{
			MetricSetFields: m.eventMapping(nets[i], &metricData{
				assetsName: assetNames,
			}),
		})
	}

	return nil
}

func getAssetNames(ctx context.Context, pc *property.Collector, net *mo.Network) (assetNames, error) {
	referenceList := append(net.Host, net.Vm...)

	var objects []mo.ManagedEntity
	if len(referenceList) > 0 {
		if err := pc.Retrieve(ctx, referenceList, []string{"name"}, &objects); err != nil {
			return assetNames{}, err
		}
	}

	outputHostNames := make([]string, 0, len(net.Host))
	outputVmNames := make([]string, 0, len(net.Vm))
	for _, ob := range objects {
		name := strings.ReplaceAll(ob.Name, ".", "_")
		switch ob.Reference().Type {
		case "HostSystem":
			outputHostNames = append(outputHostNames, name)
		case "VirtualMachine":
			outputVmNames = append(outputVmNames, name)
		}
	}

	return assetNames{
		outputVmNames:   outputVmNames,
		outputHostNames: outputHostNames,
	}, nil
}
