// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package host

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/vsphere"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func init() {
	mb.Registry.MustAddMetricSet("vsphere", "host", New,
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

// Fetch methods implements the data gathering and data conversion to the right
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
			m.Logger().Debug(fmt.Errorf("error trying to logout from vshphere: %w", err))
		}
	}()

	c := client.Client

	// Create a view of HostSystem objects.
	mgr := view.NewManager(c)

	v, err := mgr.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"HostSystem"}, true)
	if err != nil {
		return fmt.Errorf("error in CreateContainerView: %w", err)
	}

	defer func() {
		if err := v.Destroy(ctx); err != nil {
			m.Logger().Debug(fmt.Errorf("error trying to destroy view from vshphere: %w", err))
		}
	}()

	// Retrieve summary property for all hosts.
	var hst []mo.HostSystem
	err = v.Retrieve(ctx, []string{"HostSystem"}, []string{"summary"}, &hst)
	if err != nil {
		return fmt.Errorf("error in Retrieve: %w", err)
	}

	for _, hs := range hst {

		event := mapstr.M{}

<<<<<<< HEAD
		event["name"] = hs.Summary.Config.Name
		event.Put("cpu.used.mhz", hs.Summary.QuickStats.OverallCpuUsage)
		event.Put("memory.used.bytes", int64(hs.Summary.QuickStats.OverallMemoryUsage)*1024*1024)

		if hs.Summary.Hardware != nil {
			totalCPU := int64(hs.Summary.Hardware.CpuMhz) * int64(hs.Summary.Hardware.NumCpuCores)
			event.Put("cpu.total.mhz", totalCPU)
			event.Put("cpu.free.mhz", int64(totalCPU)-int64(hs.Summary.QuickStats.OverallCpuUsage))
			event.Put("memory.free.bytes", int64(hs.Summary.Hardware.MemorySize)-(int64(hs.Summary.QuickStats.OverallMemoryUsage)*1024*1024))
			event.Put("memory.total.bytes", hs.Summary.Hardware.MemorySize)
		} else {
			m.Logger().Debug("'Hardware' or 'Summary' data not found. This is either a parsing error from vsphere library, an error trying to reach host/guest or incomplete information returned from host/guest")
		}

		if hs.Summary.Host != nil {
			networkNames, err := getNetworkNames(ctx, c, hs.Summary.Host.Reference())
			if err != nil {
				m.Logger().Debugf("error trying to get network names: %s", err.Error())
			} else {
				if len(networkNames) > 0 {
					event["network_names"] = networkNames
				}
			}
		}
		reporter.Event(mb.Event{
			MetricSetFields: event,
=======
	pc := property.DefaultCollector(c)
	for i := range hst {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		assetNames, err := getAssetNames(ctx, pc, &hst[i])
		if err != nil {
			m.Logger().Errorf("Failed to retrieve object from host %s: %v", hst[i].Name, err)
		}

		metricMap, err := m.getPerfMetrics(ctx, perfManager, hst[i], metrics)
		if err != nil {
			m.Logger().Errorf("Failed to retrieve performance metrics from host %s: %v", hst[i].Name, err)
		}

		reporter.Event(mb.Event{
			MetricSetFields: m.mapEvent(hst[i], &metricData{
				perfMetrics: metricMap,
				assetNames:  assetNames,
			}),
>>>>>>> 3f44bd1f9b ([Metricbeat][vSphere] New metrics support and minor changes to existing metricsets (#40766))
		})
	}

	return nil
}

func getNetworkNames(ctx context.Context, c *vim25.Client, ref types.ManagedObjectReference) ([]string, error) {
	var outputNetworkNames []string

	pc := property.DefaultCollector(c)

<<<<<<< HEAD
	var hs mo.HostSystem
	err := pc.RetrieveOne(ctx, ref, []string{"network"}, &hs)
=======
	outputDsNames := make([]string, 0, len(hs.Datastore))
	outputVmNames := make([]string, 0, len(hs.Vm))
	for _, ob := range objects {
		name := strings.ReplaceAll(ob.Name, ".", "_")
		switch ob.Reference().Type {
		case "Datastore":
			outputDsNames = append(outputDsNames, name)
		case "VirtualMachine":
			outputVmNames = append(outputVmNames, name)
		}
	}

	// calling network explicitly because of mo.Network's ManagedEntityObject.Name does not store Network name
	// instead mo.Network.Name contains correct value of Network name
	outputNetworkNames := make([]string, 0, len(hs.Network))
	if len(hs.Network) > 0 {
		var netObjects []mo.Network
		if err := pc.Retrieve(ctx, hs.Network, []string{"name"}, &netObjects); err != nil {
			return assetNames{}, err
		}
		for _, ob := range netObjects {
			outputNetworkNames = append(outputNetworkNames, strings.ReplaceAll(ob.Name, ".", "_"))
		}
	}

	return assetNames{
		outputNetworkNames: outputNetworkNames,
		outputDsNames:      outputDsNames,
		outputVmNames:      outputVmNames,
	}, nil
}

func (m *HostMetricSet) getPerfMetrics(ctx context.Context, perfManager *performance.Manager, hst mo.HostSystem, metrics map[string]*types.PerfCounterInfo) (metricMap map[string]interface{}, err error) {
	metricMap = make(map[string]interface{})

	period := int32(m.Module().Config().Period.Seconds())
	availableMetric, err := perfManager.AvailableMetric(ctx, hst.Reference(), period)
	if err != nil {
		return nil, fmt.Errorf("failed to get available metrics: %w", err)
	}

	availableMetricByKey := availableMetric.ByKey()

	// Filter for required metrics
	var metricIDs []types.PerfMetricId
	for key, metric := range metricSet {
		if counter, ok := metrics[key]; ok {
			if _, exists := availableMetricByKey[counter.Key]; exists {
				metricIDs = append(metricIDs, types.PerfMetricId{
					CounterId: counter.Key,
					Instance:  "*",
				})
			}
		} else {
			m.Logger().Warnf("Metric %s not found", metric)
		}
	}

	spec := types.PerfQuerySpec{
		Entity:     hst.Reference(),
		MetricId:   metricIDs,
		MaxSample:  1,
		IntervalId: period,
	}

	// Query performance data
	samples, err := perfManager.Query(ctx, []types.PerfQuerySpec{spec})
>>>>>>> 3f44bd1f9b ([Metricbeat][vSphere] New metrics support and minor changes to existing metricsets (#40766))
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
