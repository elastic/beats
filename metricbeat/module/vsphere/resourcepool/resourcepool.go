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

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/vsphere"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host is defined in the module's configuration. After the
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
type PerformanceMetrics struct {
	CPUUsageAverage      int64
	ResCPUActAv1Latest   int64
	ResCPUActPk1Latest   int64
	CPUUsageMHzAverage   int64
	MemUsageAverage      int64
	MemSharedAverage     int64
	MemSwapInAverage     int64
	CPUEntitlementLatest int64
	MemEntitlementLatest int64
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

	v, err := mgr.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"ResourcePool"}, true)
	if err != nil {
		return fmt.Errorf("error in CreateContainerView: %w", err)
	}

	defer func() {
		if err := v.Destroy(ctx); err != nil {
			m.Logger().Debug(fmt.Errorf("error trying to destroy view from vshphere: %w", err))
		}
	}()

	// Retrieve summary property for all hosts.
	var rps []mo.ResourcePool
	err = v.Retrieve(ctx, []string{"ResourcePool"}, []string{"summary"}, &rps)
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

	// Define metrics to be collected
	metricNames := []string{
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

	// Define reference of structure
	var metricsVar PerformanceMetrics

	// Map metric names to structure fields
	metricMap := map[string]*int64{
		"cpu.usage.average":         &metricsVar.CPUUsageAverage,
		"rescpu.actav1.latest":      &metricsVar.ResCPUActAv1Latest,
		"rescpu.actpk1.latest":      &metricsVar.ResCPUActPk1Latest,
		"cpu.usagemhz.average":      &metricsVar.CPUUsageMHzAverage,
		"mem.usage.average":         &metricsVar.MemUsageAverage,
		"mem.shared.average":        &metricsVar.MemSharedAverage,
		"mem.swapin.average":        &metricsVar.MemSwapInAverage,
		"cpu.cpuentitlement.latest": &metricsVar.CPUEntitlementLatest,
		"mem.mementitlement.latest": &metricsVar.MemEntitlementLatest,
	}

	var spec types.PerfQuerySpec
	metricIDs := make([]types.PerfMetricId, 0, len(metricMap))

	for _, metricName := range metricNames {
		metric, exists := metrics[metricName]
		if !exists {
			m.Logger().Debug("Metric ", metricName, " not found")
			continue
		}

		metricIDs = append(metricIDs, types.PerfMetricId{
			CounterId: metric.Key,
		})
	}

	for _, rp := range rps {
		spec = types.PerfQuerySpec{
			Entity:     rp.Reference(),
			MetricId:   metricIDs,
			MaxSample:  1,
			IntervalId: 20, // right now we are only grabbing real time metrics from the performance manager
		}

		// Query performance data
		samples, err := perfManager.Query(ctx, []types.PerfQuerySpec{spec})
		if err != nil {
			m.Logger().Debug("Failed to query performance data: %v", err)
			continue
		}

		if len(samples) > 0 {
			results, err := perfManager.ToMetricSeries(ctx, samples)
			if err != nil {
				m.Logger().Debug("Failed to query performance data: %v", err)
			}

			for _, result := range results[0].Value {
				if len(result.Value) > 0 {
					if assignValue, exists := metricMap[result.Name]; exists {
						*assignValue = result.Value[0] // Assign the metric value to the variable
					}
				} else {
					m.Logger().Debug("Metric ", result.Name, ": No result found")
				}
			}
		} else {
			m.Logger().Debug("No samples returned from performance manager")
		}

		reporter.Event(mb.Event{
			MetricSetFields: m.eventMapping(rp, &metricsVar),
		})
	}

	return nil
}
