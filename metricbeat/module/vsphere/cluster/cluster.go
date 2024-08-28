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

package cluster

import (
	"context"
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/vsphere"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each network is defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("vsphere", "cluster", New,
		mb.WithHostParser(vsphere.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet.
type ClusterMetricSet struct {
	*vsphere.MetricSet
}

type assetNames struct {
	outputNtNames []string
	outputDsNames []string
	outputHsNames []string
}

// New creates a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := vsphere.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	return &ClusterMetricSet{ms}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *ClusterMetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) error {
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

	// Create a view of Cluster objects
	mgr := view.NewManager(c)

	v, err := mgr.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"ClusterComputeResource"}, true)
	if err != nil {
		return fmt.Errorf("error in CreateContainerView: %w", err)
	}

	defer func() {
		if err := v.Destroy(ctx); err != nil {
			m.Logger().Errorf("error trying to destroy view from vshphere: %w", err)
		}
	}()

	// Retrieve summary property for all Clusters
	var clt []mo.ClusterComputeResource
	err = v.Retrieve(ctx, []string{"ClusterComputeResource"}, []string{}, &clt)
	if err != nil {
		return fmt.Errorf("error in Retrieve: %w", err)
	}

	pc := property.DefaultCollector(c)
	for i := range clt {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			assetNames, err := getAssetNames(ctx, pc, &clt[i])
			if err != nil {
				m.Logger().Errorf("Failed to retrieve object from cluster %s: %v", clt[i].Name, err)
			}

			if clt[i].Configuration.DasConfig.AdmissionControlEnabled == nil {
				m.Logger().Warnf("Metric das_config.admission.control.enabled not found")
			}
			if clt[i].Configuration.DasConfig.Enabled == nil {
				m.Logger().Warnf("Metric das_config.enabled not found")
			}

			reporter.Event(mb.Event{
				MetricSetFields: m.mapEvent(clt[i], assetNames),
			})
		}
	}
	return nil

}

func getAssetNames(ctx context.Context, pc *property.Collector, cl *mo.ClusterComputeResource) (*assetNames, error) {
	referenceList := make([]types.ManagedObjectReference, 0, len(cl.Datastore)+len(cl.Host))
	referenceList = append(referenceList, cl.Datastore...)
	referenceList = append(referenceList, cl.Host...)

	outputDsNames := make([]string, 0, len(cl.Datastore))
	outputHsNames := make([]string, 0, len(cl.Host))
	if len(referenceList) > 0 {
		var objects []mo.ManagedEntity
		if err := pc.Retrieve(ctx, referenceList, []string{"name"}, &objects); err != nil {
			return nil, err
		}

		for _, ob := range objects {
			name := strings.ReplaceAll(ob.Name, ".", "_")
			switch ob.Reference().Type {
			case "Datastore":
				outputDsNames = append(outputDsNames, name)
			case "HostSystem":
				outputHsNames = append(outputHsNames, name)
			}
		}
	}

	// calling network explicitly because of mo.Network's ManagedEntityObject.Name does not store Network name
	// instead mo.Network.Name contains correct value of Network name
	outputNtNames := make([]string, 0, len(cl.Network))
	if len(cl.Network) > 0 {
		var netObjects []mo.Network
		if err := pc.Retrieve(ctx, cl.Network, []string{"name"}, &netObjects); err != nil {
			return nil, err
		}

		for _, ob := range netObjects {
			name := strings.ReplaceAll(ob.Name, ".", "_")
			outputNtNames = append(outputNtNames, name)
		}
	}

	return &assetNames{
		outputNtNames: outputNtNames,
		outputDsNames: outputDsNames,
		outputHsNames: outputHsNames,
	}, nil
}
