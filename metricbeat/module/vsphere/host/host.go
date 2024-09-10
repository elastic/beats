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
type HostMetricSet struct {
	*vsphere.MetricSet
}

// New creates a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := vsphere.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	return &HostMetricSet{ms}, nil
}

type metricData struct {
	perfMetrics map[string]interface{}
	assetNames  assetNames
	alertNames  []string
}

type assetNames struct {
	outputNetworkNames []string
	outputDsNames      []string
	outputVmNames      []string
}

// Define metrics to be collected
var metricSet = map[string]struct{}{
	"disk.capacity.usage.average": {},
	"disk.deviceLatency.average":  {},
	"disk.maxTotalLatency.latest": {},
	"disk.usage.average":          {},
	"disk.read.average":           {},
	"disk.write.average":          {},
	"net.transmitted.average":     {},
	"net.received.average":        {},
	"net.usage.average":           {},
	"net.packetsTx.summation":     {},
	"net.packetsRx.summation":     {},
	"net.errorsTx.summation":      {},
	"net.errorsRx.summation":      {},
	"net.multicastTx.summation":   {},
	"net.multicastRx.summation":   {},
	"net.droppedTx.summation":     {},
	"net.droppedRx.summation":     {},
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *HostMetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) error {
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

	// Create a view of HostSystem objects.
	mgr := view.NewManager(c)

	v, err := mgr.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"HostSystem"}, true)
	if err != nil {
		return fmt.Errorf("error in CreateContainerView: %w", err)
	}

	defer func() {
		if err := v.Destroy(ctx); err != nil {
			m.Logger().Errorf("error trying to destroy view from vSphere: %w", err)
		}
	}()

	// Retrieve summary property for all hosts.
	var hst []mo.HostSystem
	err = v.Retrieve(ctx, []string{"HostSystem"}, []string{"summary", "network", "name", "vm", "datastore", "triggeredAlarmState"}, &hst)
	if err != nil {
		return fmt.Errorf("error in Retrieve: %w", err)
	}

	// Create a performance manager
	perfManager := performance.NewManager(c)

	// Retrieve all available metrics
	metrics, err := perfManager.CounterInfoByName(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve metrics: %w", err)
	}

	// Filter for required metrics
	var metricIds []types.PerfMetricId
	for metricName := range metricSet {
		if metric, ok := metrics[metricName]; ok {
			metricIds = append(metricIds, types.PerfMetricId{CounterId: metric.Key})
		} else {
			m.Logger().Warnf("Metric %s not found", metricName)
		}
	}

	pc := property.DefaultCollector(c)
	for i := range hst {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			assetNames, err := getAssetNames(ctx, pc, &hst[i])
			if err != nil {
				m.Logger().Errorf("Failed to retrieve object from host %s: %w", hst[i].Name, err)
			}

			spec := types.PerfQuerySpec{
				Entity:     hst[i].Reference(),
				MetricId:   metricIds,
				MaxSample:  1,
				IntervalId: 20, // right now we are only grabbing real time metrics from the performance manager
			}

			// Query performance data
			samples, err := perfManager.Query(ctx, []types.PerfQuerySpec{spec})
			if err != nil {
				m.Logger().Errorf("Failed to query performance data from host %s: %v", hst[i].Name, err)
				continue
			}

			if len(samples) == 0 {
				m.Logger().Debug("No samples returned from performance manager")
				continue
			}

			results, err := perfManager.ToMetricSeries(ctx, samples)
			if err != nil {
				m.Logger().Errorf("Failed to convert performance data to metric series for host %s: %v", hst[i].Name, err)
			}

			metricMap := make(map[string]interface{})
			for _, result := range results[0].Value {
				if len(result.Value) > 0 {
					metricMap[result.Name] = result.Value[0]
					continue
				}
				m.Logger().Debugf("For host %s,Metric %v: No result found", hst[i].Name, result.Name)
			}

			alerts, err := getAlertNames(ctx, pc, hst[i].TriggeredAlarmState)
			if err != nil {
				m.Logger().Errorf("Failed to retrieve alerts from host %s: %w", hst[i].Name, err)
			}

			reporter.Event(mb.Event{
				MetricSetFields: m.mapEvent(hst[i], &metricData{perfMetrics: metricMap, assetNames: assetNames, alertNames: alerts}),
			})
		}
	}

	return nil
}

func getAssetNames(ctx context.Context, pc *property.Collector, hs *mo.HostSystem) (assetNames, error) {
	referenceList := append(hs.Datastore, hs.Vm...)

	var objects []mo.ManagedEntity
	if len(referenceList) > 0 {
		if err := pc.Retrieve(ctx, referenceList, []string{"name"}, &objects); err != nil {
			return assetNames{}, err
		}
	}

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

func getAlertNames(ctx context.Context, pc *property.Collector, triggeredAlarmState []types.AlarmState) ([]string, error) {
	var alerts []string
	for _, alarm := range triggeredAlarmState {
		if alarm.OverallStatus == "red" {
			var triggeredAlarm mo.Alarm
			err := pc.RetrieveOne(ctx, alarm.Alarm, nil, &triggeredAlarm)
			if err != nil {
				return nil, err
			}

			alerts = append(alerts, triggeredAlarm.Info.Name)
		}
	}
	return alerts, nil
}
