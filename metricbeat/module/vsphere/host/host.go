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
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

const (
	LiveInterval float64 = 20
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
}

type assetNames struct {
	outputNetworkNames   []string
	outputDatastoreNames []string
	outputVmNames        []string
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
	period := m.Module().Config().Period
	if !isValidPeriod(period.Seconds()) {
		return fmt.Errorf("invalid period %v. Please provide one of the following values: 20, 300, 1800, 7200, 86400", period)
	}

	client, err := govmomi.NewClient(ctx, m.HostURL, m.Insecure)
	if err != nil {
		return fmt.Errorf("error in NewClient: %w", err)
	}

	defer func() {
		if err := client.Logout(ctx); err != nil {
			m.Logger().Errorf("error trying to log out from vSphere: %v", err)
		}
	}()

	v, err := view.NewManager(client.Client).CreateContainerView(ctx, client.Client.ServiceContent.RootFolder, []string{"HostSystem"}, true)
	if err != nil {
		return fmt.Errorf("error in creating container view: %w", err)
	}

	defer func() {
		if err := v.Destroy(ctx); err != nil {
			m.Logger().Errorf("error trying to destroy view from vSphere: %v", err)
		}
	}()

	// Retrieve summary property for all hosts.
	var hst []mo.HostSystem
	err = v.Retrieve(ctx, []string{"HostSystem"}, []string{"summary", "network", "name", "vm", "datastore"}, &hst)
	if err != nil {
		return fmt.Errorf("error in retrieve from vsphere: %w", err)
	}

	// Create a performance manager
	perfManager := performance.NewManager(client.Client)

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

	pc := property.DefaultCollector(client.Client)
	for i := range hst {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		assetNames, err := getAssetNames(ctx, pc, &hst[i])
		if err != nil {
			m.Logger().Errorf("Failed to retrieve object from host %s: %v", hst[i].Name, err)
		}

		metricMap, err := m.getPerfMetrics(ctx, perfManager, hst[i], metricIds)
		if err != nil {
			m.Logger().Errorf("Failed to retrieve performance metrics from host %s: %v", hst[i].Name, err)
		}

		var alerts []string
		var alarmManager mo.AlarmManager
		err = client.RetrieveOne(ctx, *client.ServiceContent.AlarmManager, nil, &alarmManager)
		if err != nil {
			m.Logger().Errorf("can not retrive alarm manager from host %s: %v", hst[i].Name, err)
		} else {
			alarmStates, _ := methods.GetAlarmState(ctx, client, &types.GetAlarmState{
				This:   alarmManager.Self,
				Entity: hst[i].Self,
			})

			for _, alarm := range alarmStates.Returnval {
				if alarm.OverallStatus == "red" {
					var triggeredAlarm mo.Alarm
					err := pc.RetrieveOne(ctx, alarm.Alarm, nil, &triggeredAlarm)
					if err != nil {
						m.Logger().Errorf("can not retrive alarm from host %s: %v", hst[i].Name, err)
					}

					alerts = append(alerts, triggeredAlarm.Info.Name)
				}
			}
		}

		reporter.Event(mb.Event{
			MetricSetFields: m.mapEvent(hst[i], &metricData{perfMetrics: metricMap, assetNames: assetNames}, alerts),
		})
	}

	return nil
}

func getAssetNames(ctx context.Context, pc *property.Collector, hs *mo.HostSystem) (assetNames, error) {
	referenceList := append(hs.Datastore, hs.Vm...)

	if len(referenceList) == 0 {
		return assetNames{}, nil
	}

	var objects []mo.ManagedEntity
	if err := pc.Retrieve(ctx, referenceList, []string{"name"}, &objects); err != nil {
		return assetNames{}, err
	}

	outputDatastoreNames := make([]string, 0, len(hs.Datastore))
	outputVmNames := make([]string, 0, len(hs.Vm))
	for _, ob := range objects {
		name := strings.ReplaceAll(ob.Name, ".", "_")
		switch ob.Reference().Type {
		case "Datastore":
			outputDatastoreNames = append(outputDatastoreNames, name)
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
		outputNetworkNames:   outputNetworkNames,
		outputDatastoreNames: outputDatastoreNames,
		outputVmNames:        outputVmNames,
	}, nil
}

func (m *HostMetricSet) getPerfMetrics(ctx context.Context, perfManager *performance.Manager, hst mo.HostSystem, metricIds []types.PerfMetricId) (map[string]interface{}, error) {
	metricMap := make(map[string]interface{})
	summary, err := perfManager.ProviderSummary(ctx, hst.Reference())
	if err != nil {
		return metricMap, fmt.Errorf("failed to get summary: %w", err)
	}

	period := m.Module().Config().Period
	refreshRate := int32(period.Seconds())
	if period.Seconds() == LiveInterval {
		if summary.CurrentSupported {
			refreshRate = summary.RefreshRate
			if int32(period.Seconds()) != refreshRate {
				m.Logger().Warnf("User-provided period %v does not match system's refresh rate %v. Risk of data duplication. Consider adjusting period.", period, refreshRate)
			}
		} else {
			m.Logger().Warnf("Live data collection not supported. Use one of the system's historical interval (300, 1800, 7200, 86400). Risk of data duplication. Consider adjusting period.")
		}
	}

	spec := types.PerfQuerySpec{
		Entity:     hst.Reference(),
		MetricId:   metricIds,
		MaxSample:  1,
		IntervalId: refreshRate,
	}

	// Query performance data
	samples, err := perfManager.Query(ctx, []types.PerfQuerySpec{spec})
	if err != nil {
		return metricMap, fmt.Errorf("failed to query performance data: %w", err)
	}

	if len(samples) == 0 {
		m.Logger().Debug("No samples returned from performance manager")
		return metricMap, nil
	}

	results, err := perfManager.ToMetricSeries(ctx, samples)
	if err != nil {
		return metricMap, fmt.Errorf("failed to convert performance data to metric series: %w", err)
	}

	for _, result := range results[0].Value {
		if len(result.Value) > 0 {
			metricMap[result.Name] = result.Value[0]
			continue
		}
		m.Logger().Debugf("For host %s, Metric %s: No result found", hst.Name, result.Name)
	}

	return metricMap, nil
}

func isValidPeriod(period float64) bool {
	switch period {
	case LiveInterval, 300, 1800, 7200, 86400:
		return true
	}
	return false
}
