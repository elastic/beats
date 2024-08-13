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
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/vsphere"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
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

type metricData struct {
	perfMetrics map[string]interface{}
	assetsName  assetNames
}

type assetNames struct {
	outputNetworkNames []string
	outputDsNames      []string
	outputVmNames      []string
}

// Define metrics to be collected
var metricNames = []string{
	"disk.capacity.usage.average",
	"disk.deviceLatency.average",
	"disk.maxTotalLatency.latest",
	"disk.usage.average",
	"disk.read.average",
	"disk.write.average",
	"net.transmitted.average",
	"net.received.average",
	"net.usage.average",
	"net.packetsTx.summation",
	"net.packetsRx.summation",
	"net.errorsTx.summation",
	"net.errorsRx.summation",
	"net.multicastTx.summation",
	"net.multicastRx.summation",
	"net.droppedTx.summation",
	"net.droppedRx.summation",
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
	err = v.Retrieve(ctx, []string{"HostSystem"}, []string{"summary", "network", "name", "vm", "datastore"}, &hst)
	if err != nil {
		return fmt.Errorf("error in Retrieve: %w", err)
	}

	// Create a performance manager
	perfManager := performance.NewManager(c)

	// Retrieve metric IDs for the specified metric names
	metrics, err := perfManager.CounterInfoByName(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve metrics: %w", err)
	}

	// Retrieve only the required metrics
	requiredMetrics := make(map[string]*types.PerfCounterInfo)

	for _, name := range metricNames {
		metric, exists := metrics[name]
		if !exists {
			m.Logger().Warnf("Metric %s not found", name)
			continue
		}
		requiredMetrics[name] = metric
	}

	metricIDs := make([]types.PerfMetricId, 0, len(requiredMetrics))
	for _, metric := range requiredMetrics {
		metricIDs = append(metricIDs, types.PerfMetricId{
			CounterId: metric.Key,
		})
	}
	metricMap := map[string]interface{}{}

	pc := property.DefaultCollector(c)
	for _, hs := range hst {
		assetNames, err := getAssetNames(ctx, pc, &hs)
		if err != nil {
			m.Logger().Errorf("Failed to retrieve object from host: %w", err)
		}

		spec := types.PerfQuerySpec{
			Entity:     hs.Reference(),
			MetricId:   metricIDs,
			MaxSample:  1,
			IntervalId: 20, // right now we are only grabbing real time metrics from the performance manager
		}

		// Query performance data
		samples, err := perfManager.Query(ctx, []types.PerfQuerySpec{spec})
		if err != nil {
			m.Logger().Errorf("Failed to query performance data: %v", err)
			continue
		}

		if len(samples) == 0 {
			m.Logger().Debug("No samples returned from performance manager")
			continue
		}

		results, err := perfManager.ToMetricSeries(ctx, samples)
		if err != nil {
			m.Logger().Errorf("Failed to convert performance data: %v", err)
		}

		for _, result := range results[0].Value {
			if len(result.Value) > 0 {
				metricMap[result.Name] = result.Value[0]
				continue
			}
			m.Logger().Debugf("Metric %v: No result found", result.Name)
		}

		reporter.Event(mb.Event{
			MetricSetFields: m.eventMapping(hs, &metricData{
				perfMetrics: metricMap,
				assetsName:  *assetNames,
			}),
		})
	}

	return nil
}

func getAssetNames(ctx context.Context, pc *property.Collector, hs *mo.HostSystem) (*assetNames, error) {
	referenceList := make([]types.ManagedObjectReference, 0, len(hs.Datastore)+len(hs.Vm)+len(hs.Network))
	referenceList = append(referenceList, hs.Datastore...)
	referenceList = append(referenceList, hs.Vm...)
	referenceList = append(referenceList, hs.Network...)

	var objects []mo.ManagedEntity
	if len(referenceList) > 0 {
		if err := pc.Retrieve(ctx, referenceList, []string{"name"}, &objects); err != nil {
			return nil, err
		}
	}

	outputDsNames := make([]string, 0, len(hs.Datastore))
	outputVmNames := make([]string, 0, len(hs.Vm))
	outputNetworkNames := make([]string, 0, len(hs.Network))
	for _, ob := range objects {
		name := strings.ReplaceAll(ob.Name, ".", "_")
		switch ob.Reference().Type {
		case "Datastore":
			outputDsNames = append(outputDsNames, name)
		case "VirtualMachine":
			outputVmNames = append(outputVmNames, name)
		case "Network":
			outputNetworkNames = append(outputNetworkNames, name)
		}
	}

	return &assetNames{
		outputNetworkNames: outputNetworkNames,
		outputDsNames:      outputDsNames,
		outputVmNames:      outputVmNames,
	}, nil
}
