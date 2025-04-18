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

package datastore

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
	mb.Registry.MustAddMetricSet("vsphere", "datastore", New,
		mb.WithHostParser(vsphere.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet.
type DataStoreMetricSet struct {
	*vsphere.MetricSet
}

// New creates a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := vsphere.NewMetricSet(base)
	if err != nil {
		return nil, err
	}

	security.WarnIfInsecure(ms.Logger(), "datastore", ms.Insecure)
	return &DataStoreMetricSet{ms}, nil
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
	outputVmNames   []string
	outputHostNames []string
}

// Define metrics to be collected
var metricSet = map[string]struct{}{
	"datastore.read.average":      {},
	"datastore.write.average":     {},
	"disk.capacity.latest":        {},
	"disk.capacity.usage.average": {},
	"disk.provisioned.latest":     {},
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *DataStoreMetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) error {
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

	// Create a view of Datastore objects
	mgr := view.NewManager(c)

	v, err := mgr.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"Datastore"}, true)
	if err != nil {
		return fmt.Errorf("error in creating container view: %w", err)
	}

	defer func() {
		if err := v.Destroy(ctx); err != nil {
			m.Logger().Debugf("error trying to destroy view from vSphere: %v", err)
		}
	}()

	// Retrieve summary property for all datastores
	var dst []mo.Datastore
	err = v.Retrieve(ctx, []string{"Datastore"}, []string{"summary", "host", "vm", "overallStatus", "triggeredAlarmState"}, &dst)
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

	pc := property.DefaultCollector(client.Client)
	for i := range dst {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		assetNames, err := getAssetNames(ctx, pc, &dst[i])
		if err != nil {
			m.Logger().Errorf("Failed to retrieve object from datastore %s: %v", dst[i].Name, err)
		}

		metricMap, err := m.getPerfMetrics(ctx, perfManager, dst[i], metrics)
		if err != nil {
			m.Logger().Errorf("Failed to retrieve performance metrics from datastore %s: %v", dst[i].Name, err)
		}

		triggeredAlarm, err := getTriggeredAlarm(ctx, pc, dst[i].TriggeredAlarmState)
		if err != nil {
			m.Logger().Errorf("Failed to retrieve alerts from datastore %s: %w", dst[i].Name, err)
		}

		reporter.Event(mb.Event{
			MetricSetFields: m.mapEvent(dst[i], &metricData{
				perfMetrics:     metricMap,
				triggeredAlarms: triggeredAlarm,
				assetNames:      assetNames,
			}),
		})
	}

	return nil
}

func getAssetNames(ctx context.Context, pc *property.Collector, ds *mo.Datastore) (assetNames, error) {
	outputVmNames := make([]string, 0, len(ds.Vm))
	if len(ds.Vm) > 0 {
		var objects []mo.ManagedEntity
		if err := pc.Retrieve(ctx, ds.Vm, []string{"name"}, &objects); err != nil {
			return assetNames{}, err
		}

		for _, ob := range objects {
			if ob.Reference().Type == "VirtualMachine" {
				name := strings.ReplaceAll(ob.Name, ".", "_")
				outputVmNames = append(outputVmNames, name)
			}
		}
	}

	// calling Host explicitly because of mo.Datastore.Host has types.DatastoreHostMount instead of mo.ManagedEntity
	outputHostNames := make([]string, 0, len(ds.Host))
	if len(ds.Host) > 0 {
		hsRefs := make([]types.ManagedObjectReference, 0, len(ds.Host))
		for _, obj := range ds.Host {
			if obj.Key.Type == "HostSystem" {
				hsRefs = append(hsRefs, obj.Key)
			}
		}

		// Retrieve Host names
		var hosts []mo.HostSystem
		if len(hsRefs) > 0 {
			err := pc.Retrieve(ctx, hsRefs, []string{"name"}, &hosts)
			if err != nil {
				return assetNames{}, err
			}
		}

		for _, host := range hosts {
			name := strings.ReplaceAll(host.Name, ".", "_")
			outputHostNames = append(outputHostNames, name)
		}
	}

	return assetNames{
		outputHostNames: outputHostNames,
		outputVmNames:   outputVmNames,
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

func (m *DataStoreMetricSet) getPerfMetrics(ctx context.Context, perfManager *performance.Manager, dst mo.Datastore, metrics map[string]*types.PerfCounterInfo) (metricMap map[string]interface{}, err error) {
	metricMap = make(map[string]interface{})

	period := int32(m.Module().Config().Period.Seconds())
	availableMetric, err := perfManager.AvailableMetric(ctx, dst.Reference(), period)
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
		Entity:     dst.Reference(),
		MetricId:   metricIDs,
		MaxSample:  1,
		IntervalId: period, // using refreshRate as interval
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
		m.Logger().Debugf("For datastore %s, Metric %s: No result found", dst.Name, result.Name)
	}

	return metricMap, nil
}
