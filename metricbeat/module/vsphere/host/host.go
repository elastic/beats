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

// Structure to hold performance metrics values
type PerformanceMetrics struct {
	NetUsage                int64
	NetDroppedTransmitted   int64
	NetDroppedReceived      int64
	NetMulticastTransmitted int64
	NetMulticastReceived    int64
	NetErrorsTransmitted    int64
	NetErrorsReceived       int64
	NetPacketTransmitted    int64
	NetPacketReceived       int64
	NetReceived             int64
	NetTransmitted          int64
	DiskWrite               int64
	DiskRead                int64
	DiskUsage               int64
	DiskMaxTotalLatency     int64
	DiskDeviceLatency       int64
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
	err = v.Retrieve(ctx, []string{"HostSystem"}, []string{"summary", "network"}, &hst)
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

	// Define reference of structure
	var metricsVar PerformanceMetrics

	// Map metric names to struture	fields
	metricMap := map[string]*int64{
		"disk.deviceLatency.average":  &metricsVar.DiskDeviceLatency,
		"disk.maxTotalLatency.latest": &metricsVar.DiskMaxTotalLatency,
		"net.usage.average":           &metricsVar.NetUsage,
		"disk.usage.average":          &metricsVar.DiskUsage,
		"disk.read.average":           &metricsVar.DiskRead,
		"disk.write.average":          &metricsVar.DiskWrite,
		"net.transmitted.average":     &metricsVar.NetTransmitted,
		"net.received.average":        &metricsVar.NetReceived,
		"net.packetsTx.summation":     &metricsVar.NetPacketTransmitted,
		"net.packetsRx.summation":     &metricsVar.NetPacketReceived,
		"net.errorsTx.summation":      &metricsVar.NetErrorsTransmitted,
		"net.errorsRx.summation":      &metricsVar.NetErrorsReceived,
		"net.multicastTx.summation":   &metricsVar.NetMulticastTransmitted,
		"net.multicastRx.summation":   &metricsVar.NetMulticastReceived,
		"net.droppedTx.summation":     &metricsVar.NetDroppedTransmitted,
		"net.droppedRx.summation":     &metricsVar.NetDroppedReceived,
	}

	// Map metric IDs to metric names
	metricNamesById := make(map[int32]string)
	for name, metric := range metrics {
		metricNamesById[metric.Key] = name
	}

	var spec types.PerfQuerySpec
	metricIDs := make([]types.PerfMetricId, 0, len(metricNames))

	for _, metricName := range metricNames {
		metric, exists := metrics[metricName]
		if !exists {
			m.Logger().Debug("Metric %v not found", metricName)
			continue
		}

		metricIDs = append(metricIDs, types.PerfMetricId{
			CounterId: metric.Key,
		})
	}

	pc := property.DefaultCollector(c)
	for _, hs := range hst {
		var networkRefs []types.ManagedObjectReference
		for _, obj := range hs.Network {
			if obj.Type == "Network" {
				networkRefs = append(networkRefs, obj)
			}
		}

		var nets []mo.Network
		if len(networkRefs) > 0 {
			err = pc.Retrieve(ctx, networkRefs, []string{"name"}, &nets)
			if err != nil {
				m.Logger().Errorf("error retrieving network from host: %v", err)
			}
		}

		var outputNetworkNames []string
		for _, net := range nets {
			name := strings.Replace(net.Name, ".", "_", -1)
			outputNetworkNames = append(outputNetworkNames, name)
		}

		spec = types.PerfQuerySpec{
			Entity:     hs.Reference(),
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
			entityMetrics, ok := samples[0].(*types.PerfEntityMetric)
			if !ok {
				m.Logger().Debug("Unexpected metric type")
				continue
			}

			for _, value := range entityMetrics.Value {
				metricSeries, ok := value.(*types.PerfMetricIntSeries)
				if !ok {
					m.Logger().Debug("Unexpected metric series type")
					continue
				}

				if len(metricSeries.Value) > 0 {
					metricName := metricNamesById[metricSeries.Id.CounterId]
					if assignValue, exists := metricMap[metricName]; exists {
						*assignValue = metricSeries.Value[0] // Assign the metric value to the variable
					}
				} else {
					m.Logger().Debug("Metric %v: No result found\n", metricNamesById[metricSeries.Id.CounterId])
				}
			}
		} else {
			m.Logger().Debug("No samples returned from performance manager")
		}

		reporter.Event(mb.Event{
			MetricSetFields: m.eventMapping(hs, &metricsVar, outputNetworkNames),
		})
	}

	return nil
}
