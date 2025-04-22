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

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/vsphere"
	vSphereClientUtil "github.com/elastic/beats/v7/metricbeat/module/vsphere/client"
	"github.com/elastic/beats/v7/metricbeat/module/vsphere/security"

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

	security.WarnIfInsecure(ms.Logger(), "host", ms.Insecure)
	return &HostMetricSet{ms}, nil
}

type triggeredAlarm struct {
	Name          string      `json:"name"`
	ID            string      `json:"id"`
	Status        string      `json:"status"`
	TriggeredTime common.Time `json:"triggered_time"`
	Description   string      `json:"description"`
	EntityName    string      `json:"entity_name"`
}

type metricData struct {
	perfMetrics     map[string]interface{}
	assetNames      assetNames
	triggeredAlarms []triggeredAlarm
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
		err := vSphereClientUtil.Logout(ctx, client)

		if err != nil {
			m.Logger().Errorf("error trying to logout from vSphere: %v", err)
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
			m.Logger().Errorf("error trying to destroy view from vSphere: %v", err)
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

		triggeredAlarm, err := getTriggeredAlarm(ctx, pc, hst[i].TriggeredAlarmState)
		if err != nil {
			m.Logger().Errorf("Failed to retrieve triggered alarms from host %s: %w", hst[i].Name, err)
		}

		reporter.Event(mb.Event{
			MetricSetFields: m.mapEvent(hst[i], &metricData{
				perfMetrics:     metricMap,
				triggeredAlarms: triggeredAlarm,
				assetNames:      assetNames,
			}),
		})
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

func getTriggeredAlarm(ctx context.Context, pc *property.Collector, triggeredAlarmState []types.AlarmState) ([]triggeredAlarm, error) {
	var triggeredAlarms []triggeredAlarm
	for _, alarmState := range triggeredAlarmState {
		var triggeredAlarm triggeredAlarm
		var alarm mo.Alarm
		err := pc.RetrieveOne(ctx, alarmState.Alarm, nil, &alarm)
		if err != nil {
			return nil, err
		}
		triggeredAlarm.Name = alarm.Info.Name

		var entityName string
		if alarmState.Entity.Type == "Network" {
			var entity mo.Network
			if err := pc.RetrieveOne(ctx, alarmState.Entity, []string{"name"}, &entity); err != nil {
				return nil, err
			}

			entityName = entity.Name
		} else {
			var entity mo.ManagedEntity
			if err := pc.RetrieveOne(ctx, alarmState.Entity, []string{"name"}, &entity); err != nil {
				return nil, err
			}

			entityName = entity.Name
		}
		triggeredAlarm.EntityName = entityName

		triggeredAlarm.Description = alarm.Info.Description
		triggeredAlarm.ID = alarmState.Key
		triggeredAlarm.Status = string(alarmState.OverallStatus)
		triggeredAlarm.TriggeredTime = common.Time(alarmState.Time)

		triggeredAlarms = append(triggeredAlarms, triggeredAlarm)
	}

	return triggeredAlarms, nil
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
	if err != nil {
		if strings.Contains(err.Error(), "ServerFaultCode: A specified parameter was not correct: querySpec.interval") {
			return metricMap, fmt.Errorf("failed to query performance data: use one of the system's supported interval. consider adjusting period: %w", err)
		}

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

	if len(results) == 0 {
		m.Logger().Debug("No results returned from metric series conversion")
		return metricMap, nil
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
