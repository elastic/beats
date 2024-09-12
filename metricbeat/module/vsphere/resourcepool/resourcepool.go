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
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/vsphere"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each resourcepool is defined in the module's configuration. After the
// MetricSet has been created, Fetch will be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("vsphere", "resourcepool", New,
		mb.WithHostParser(vsphere.HostParser),
		mb.DefaultMetricSet(),
	)
}

// ResourcePoolMetricSet type defines all fields of the MetricSet.
type ResourcePoolMetricSet struct {
	*vsphere.MetricSet
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := vsphere.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	return &ResourcePoolMetricSet{ms}, nil
}

// metricData holds performance metrics values.
type metricData struct {
	assetNames assetNames
}

type assetNames struct {
	outputVmNames []string
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *ResourcePoolMetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	client, err := govmomi.NewClient(ctx, m.HostURL, m.Insecure)
	if err != nil {
		return fmt.Errorf("error in NewClient: %w", err)
	}

	defer func() {
		if err := client.Logout(ctx); err != nil {
			m.Logger().Errorf("error trying to logout from vSphere: %v", err)
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
			m.Logger().Errorf("error trying to destroy view from vSphere: %v", err)
		}
	}()

	// Retrieve property for all ResourcePools.
	var rps []mo.ResourcePool
	err = v.Retrieve(ctx, []string{"ResourcePool"}, []string{"name", "overallStatus", "vm", "summary"}, &rps)
	if err != nil {
		return fmt.Errorf("error in Retrieve: %w", err)
	}

	pc := property.DefaultCollector(c)
	for i := range rps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			assetNames, err := getAssetNames(ctx, pc, &rps[i])
			if err != nil {
				m.Logger().Errorf("Failed to retrieve object from resource pool %s: %v", rps[i].Name, err)
			}

			reporter.Event(mb.Event{
				MetricSetFields: m.mapEvent(rps[i], &metricData{assetNames: assetNames}),
			})
		}
	}

	return nil
}

func getAssetNames(ctx context.Context, pc *property.Collector, rp *mo.ResourcePool) (assetNames, error) {
	if len(rp.Vm) == 0 {
		return assetNames{}, nil
	}

	var objects []mo.ManagedEntity
	err := pc.Retrieve(ctx, rp.Vm, []string{"name"}, &objects)
	if err != nil {
		return assetNames{}, err
	}

	outputVmNames := make([]string, 0, len(objects))
	for _, ob := range objects {
		if ob.Reference().Type == "VirtualMachine" {
			name := strings.ReplaceAll(ob.Name, ".", "_")
			outputVmNames = append(outputVmNames, name)
		}
	}

	return assetNames{outputVmNames: outputVmNames}, nil
}
