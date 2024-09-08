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

package alert

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/vsphere"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each network is defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("vsphere", "alert", New,
		mb.WithHostParser(vsphere.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet.
type AlertMetricSet struct {
	*vsphere.MetricSet
}

type alert struct {
	name       string
	entityName string
	status     string
}

// New creates a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := vsphere.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	return &AlertMetricSet{ms}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *AlertMetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	client, err := govmomi.NewClient(ctx, m.HostURL, m.Insecure)
	if err != nil {
		return fmt.Errorf("error in NewClient: %w", err)
	}
	defer func() {
		if err := client.Logout(ctx); err != nil {
			m.Logger().Debug(fmt.Errorf("error trying to logout from vSphere: %w", err))
		}
	}()

	pc := property.DefaultCollector(client.Client)

	// Retrieve the root folder
	folder := object.NewFolder(client.Client, client.ServiceContent.RootFolder)

	// Retrieve alarms
	var folderObj mo.Folder
	err = client.RetrieveOne(ctx, folder.Reference(), []string{"triggeredAlarmState"}, &folderObj)
	if err != nil {
		log.Fatalf("Error retrieving triggered alarms: %v", err)
	}

	for _, alarmState := range folderObj.TriggeredAlarmState {
		var alarm mo.Alarm
		pc.Retrieve(ctx, []types.ManagedObjectReference{alarmState.Alarm.Reference()}, nil, &alarm)

		if alarmState.OverallStatus == types.ManagedEntityStatusRed {
			var entity mo.ManagedEntity
			pc.RetrieveOne(ctx, alarmState.Entity, nil, &entity)

			entityName, err := getAssetNames(ctx, pc, alarmState.Entity)
			if err != nil {
				return fmt.Errorf("failed to retrieve managed entity name: %w", err)
			}

			reporter.Event(mb.Event{
				MetricSetFields: m.mapEvent(alert{
					name:       alarm.Info.Name,
					entityName: entityName,
					status:     string(alarmState.OverallStatus),
				}),
			})

		}
	}

	return nil

}

func getAssetNames(ctx context.Context, pc *property.Collector, entity types.ManagedObjectReference) (string, error) {

	if entity.Type != "Network" {
		var object mo.ManagedEntity
		if err := pc.RetrieveOne(ctx, entity, []string{"name"}, &object); err != nil {
			return "", fmt.Errorf("failed to retrieve managed entities: %w", err)
		}
		return strings.ReplaceAll(object.Name, ".", "_"), nil
	}

	// calling network explicitly because of mo.Network's ManagedEntityObject.Name does not store Network name
	// instead mo.Network.Name contains correct value of Network name
	var netObject mo.Network
	if err := pc.RetrieveOne(ctx, entity, []string{"name"}, &netObject); err != nil {
		return "", fmt.Errorf("failed to retrieve network objects: %w", err)
	}

	return strings.ReplaceAll(netObject.Name, ".", "_"), nil
}
