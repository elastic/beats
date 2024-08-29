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
	return &DataStoreMetricSet{ms}, nil
}

type metricData struct {
	perfMetrics map[string]interface{}
	assetNames  assetNames
}

type assetNames struct {
	outputVmNames []string
	outputHostNames []string
}

// Define metrics to be collected
var metricSet = map[string]struct{}{
	"datastore.read.average":              {},
	"datastore.write.average":             {},
	"datastore.datastoreIops.average":     {},
	"datastore.totalReadLatency.average":  {},
	"datastore.totalWriteLatency.average": {},
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
		if err := client.Logout(ctx); err != nil {
			m.Logger().Debugf("error trying to log out from vSphere: %w", err)
		}
	}()

	c := client.Client

	// Create a view of Datastore objects
	mgr := view.NewManager(c)

	v, err := mgr.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"Datastore"}, true)
	if err != nil {
		return fmt.Errorf("error in CreateContainerView: %w", err)
	}

	defer func() {
		if err := v.Destroy(ctx); err != nil {
			m.Logger().Debugf("error trying to destroy view from vSphere: %w", err)
		}
	}()

	// Retrieve summary property for all datastores
	var dst []mo.Datastore
	err = v.Retrieve(ctx, []string{"Datastore"}, []string{"summary", "host", "vm", "overallStatus"}, &dst)
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
	for i := range dst {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			assetNames, err := getAssetNames(ctx, pc, &dst[i])
			if err != nil {
				m.Logger().Errorf("Failed to retrieve object from host %s: %w", dst[i].Name, err)
				continue
			}

			spec := types.PerfQuerySpec{
				Entity:     dst[i].Reference(),
				MetricId:   metricIds,
				MaxSample:  1,
				IntervalId: 20, // right now we are only grabbing real time metrics from the performance manager
			}

			// Query performance data
			samples, err := perfManager.Query(ctx, []types.PerfQuerySpec{spec})
			if err != nil {
				m.Logger().Debugf("Failed to query performance data for host %s: %v", dst[i].Name, err)
				continue
			}

			if len(samples) == 0 {
				m.Logger().Debugf("No samples returned from performance manager")
				continue
			}

			results, err := perfManager.ToMetricSeries(ctx, samples)
			if err != nil {
				m.Logger().Debugf("Failed to query performance data to metric series for host %s: %v", dst[i].Name, err)
				continue
			}

			metricMap := make(map[string]interface{})
			for _, result := range results[0].Value {
				if len(result.Value) > 0 {
					metricMap[result.Name] = result.Value[0]
					continue
				}
				m.Logger().Debugf("For host %s,Metric %v: No result found", dst[i].Name, result.Name)
			}

			reporter.Event(mb.Event{
				MetricSetFields: m.mapEvent(dst[i], &metricData{
					perfMetrics: metricMap,
					assetNames:  *assetNames,
				}),
			})
		}
	}

	return nil
}

func getAssetNames(ctx context.Context, pc *property.Collector, ds *mo.Datastore) (*assetNames, error) {

	outputVmNames := make([]string, 0, len(ds.Vm))
	if len(ds.Vm) > 0 {
		var objects []mo.ManagedEntity
		if err := pc.Retrieve(ctx, ds.Vm, []string{"name"}, &objects); err != nil {
			return nil, err
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
				return nil, err
			}
		}

		for _, host := range hosts {
			name := strings.ReplaceAll(host.Name, ".", "_")
			outputHostNames = append(outputHostNames, name)
		}
	}

	return &assetNames{
		outputHostNames: outputHostNames,
		outputVmNames: outputVmNames,
	}, nil
}
