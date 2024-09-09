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

package datastorecluster

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
// the MetricSet for each network is defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.

func init() {
	mb.Registry.MustAddMetricSet("vsphere", "datastorecluster", New,
		mb.WithHostParser(vsphere.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet.
type DatastoreClusterMetricSet struct {
	*vsphere.MetricSet
}

// New creates a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := vsphere.NewMetricSet(base)
	if err != nil {
		return nil, fmt.Errorf("failed to create vSphere metricset: %w", err)
	}
	return &DatastoreClusterMetricSet{ms}, nil
}

type metricData struct {
	assetNames assetNames
}

type assetNames struct {
	outputDsNames []string
}

func (m *DatastoreClusterMetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	client, err := govmomi.NewClient(ctx, m.HostURL, m.Insecure)
	if err != nil {
		return fmt.Errorf("error in NewClient: %w", err)
	}

	defer func() {
		if err := client.Logout(ctx); err != nil {
			m.Logger().Errorf("error trying to logout from vSphere: %w", err)
		}
	}()

	c := client.Client

	v, err := view.NewManager(c).CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"StoragePod"}, true)
	if err != nil {
		return fmt.Errorf("error in creating container view: %w", err)
	}

	defer func() {
		if err := v.Destroy(ctx); err != nil {
			m.Logger().Errorf("error trying to destroy view from vSphere: %w", err)
		}
	}()

	var datastoreCluster []mo.StoragePod
	err = v.Retrieve(ctx, []string{"StoragePod"}, []string{"name", "summary", "childEntity"}, &datastoreCluster)
	if err != nil {
		return fmt.Errorf("error in retrieve from vsphere: %w", err)
	}

	pc := property.DefaultCollector(c)
	for i := range datastoreCluster {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		assetNames, err := getAssetNames(ctx, pc, &datastoreCluster[i])
		if err != nil {
			m.Logger().Errorf("Failed to retrieve object from host %s: %w", datastoreCluster[i].Name, err)
		}

		reporter.Event(mb.Event{MetricSetFields: m.mapEvent(datastoreCluster[i], &metricData{assetNames: assetNames})})
	}

	return nil
}

func getAssetNames(ctx context.Context, pc *property.Collector, dsc *mo.StoragePod) (assetNames, error) {
	var objects []mo.ManagedEntity
	if len(dsc.ChildEntity) > 0 {
		if err := pc.Retrieve(ctx, dsc.ChildEntity, []string{"name"}, &objects); err != nil {
			return assetNames{}, err
		}
	}

	outputDsNames := make([]string, 0)
	for _, ob := range objects {
		if ob.Reference().Type == "Datastore" {
			name := strings.ReplaceAll(ob.Name, ".", "_")
			outputDsNames = append(outputDsNames, name)
		}
	}

	return assetNames{
		outputDsNames: outputDsNames,
	}, nil
}
