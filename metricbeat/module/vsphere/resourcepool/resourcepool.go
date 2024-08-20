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

package resourcepool

import (
	"context"
	"fmt"
	"strings"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/vsphere"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each resourcepool is defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("vsphere", "resourcepool", New,
		mb.WithHostParser(vsphere.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet.
type MetricSet struct {
	*vsphere.MetricSet
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := vsphere.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{ms}, nil
}

// Structure to hold performance metrics values
type metricData struct {
	perfMetrics map[string]interface{}
	assetsName  assetNames
}

type assetNames struct {
	outputVmNames []string
}

// Define metrics to be collected
var metricNames = []string{
	"cpu.usage.average",
	"rescpu.actav1.latest",
	"rescpu.actpk1.latest",
	"cpu.usagemhz.average",
	"mem.usage.average",
	"mem.shared.average",
	"mem.swapin.average",
	"cpu.cpuentitlement.latest",
	"mem.mementitlement.latest",
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
			m.Logger().Errorf("error trying to log out from vSphere: %w", err)
		}
	}()

	c := client.Client

	// Create a view of ResourcePool objects.
	mgr := view.NewManager(c)

	v, err := mgr.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"ResourcePool"}, true)
	if err != nil {
		return fmt.Errorf("error in CreateContainerView: %w", err)
	}

	defer func() {
		if err := v.Destroy(ctx); err != nil {
			m.Logger().Errorf("error trying to destroy view from vSphere: %w", err)
		}
	}()

	// Retrieve property for all ResourcePools.
	var rps []mo.ResourcePool
	err = v.Retrieve(ctx, []string{"ResourcePool"}, []string{"name", "summary", "overallStatus", "vm"}, &rps)
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

	var spec types.PerfQuerySpec

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
	pc := property.DefaultCollector(c)
	for i := range rps {

		metricMap := map[string]interface{}{}

		assetNames, err := getAssetNames(ctx, pc, &rps[i])
		if err != nil {
			m.Logger().Errorf("Failed to retrieve object from resource pool %s: %w", rps[i].Name, err)
		}

		spec = types.PerfQuerySpec{
			Entity:     rps[i].Reference(),
			MetricId:   metricIDs,
			MaxSample:  1,
			IntervalId: 20, // right now we are only grabbing real time metrics from the performance manager
		}

		// Query performance data
		samples, err := perfManager.Query(ctx, []types.PerfQuerySpec{spec})
		if err != nil {
			m.Logger().Errorf("Failed to query performance data from resource pool %s: %v", rps[i].Name, err)
			continue
		}

		if len(samples) == 0 {
			m.Logger().Debug("No samples returned from performance manager")
			continue
		}

		results, err := perfManager.ToMetricSeries(ctx, samples)
		if err != nil {
			m.Logger().Errorf("Failed to convert performance data to metric series for resource pool %s: %v", rps[i].Name, err)
		}

		for _, result := range results[0].Value {
			if len(result.Value) > 0 {
				metricMap[result.Name] = result.Value[0]
				continue
			}
			m.Logger().Debugf("For resource pool %s,Metric %v: No result found", rps[i].Name, result.Name)
		}

		reporter.Event(mb.Event{
			MetricSetFields: m.eventMapping(rps[i], &metricData{
				perfMetrics: metricMap,
				assetsName:  assetNames,
			}),
		})
	}

	return nil
}

func getAssetNames(ctx context.Context, pc *property.Collector, rp *mo.ResourcePool) (assetNames, error) {
	referenceList := append([]types.ManagedObjectReference{}, rp.Vm...)

	var objects []mo.ManagedEntity
	if len(referenceList) > 0 {
		if err := pc.Retrieve(ctx, referenceList, []string{"name"}, &objects); err != nil {
			return assetNames{}, err
		}
	}

	outputVmNames := make([]string, 0, len(rp.Vm))
	for _, ob := range objects {
		name := strings.ReplaceAll(ob.Name, ".", "_")
		switch ob.Reference().Type {
		case "VirtualMachine":
			outputVmNames = append(outputVmNames, name)
		}
	}

	return assetNames{
		outputVmNames: outputVmNames,
	}, nil
}
