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

package virtualmachine

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/vsphere"
	vSphereClientUtil "github.com/elastic/beats/v7/metricbeat/module/vsphere/client"
	"github.com/elastic/beats/v7/metricbeat/module/vsphere/security"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func init() {
	mb.Registry.MustAddMetricSet("vsphere", "virtualmachine", New,
		mb.WithHostParser(vsphere.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet.
type MetricSet struct {
	*vsphere.MetricSet
	GetCustomFields bool
}

type triggeredAlarm struct {
	Name          string      `json:"name"`
	ID            string      `json:"id"`
	Status        string      `json:"status"`
	TriggeredTime common.Time `json:"triggered_time"`
	Description   string      `json:"description"`
	EntityName    string      `json:"entity_name"`
}

type VMData struct {
	VM              mo.VirtualMachine
	HostID          string
	HostName        string
	NetworkNames    []string
	DatastoreNames  []string
	CustomFields    mapstr.M
	Snapshots       []VMSnapshotData
	triggeredAlarms []triggeredAlarm
}

type VMSnapshotData struct {
	ID          int32                          `json:"id"`
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	CreateTime  common.Time                    `json:"createtime"`
	State       types.VirtualMachinePowerState `json:"state"`
}

// New creates a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := vsphere.NewMetricSet(base)
	if err != nil {
		return nil, err
	}

	config := struct {
		GetCustomFields bool `config:"get_custom_fields"`
	}{
		GetCustomFields: false,
	}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	security.WarnIfInsecure(ms.Logger(), "virtualmachine", ms.Insecure)
	return &MetricSet{
		MetricSet:       ms,
		GetCustomFields: config.GetCustomFields,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	client, err := govmomi.NewClient(ctx, m.HostURL, m.Insecure)
	if err != nil {
		return fmt.Errorf("virtualmachine: error in NewClient: %w", err)
	}

	defer func() {
		err := vSphereClientUtil.Logout(ctx, client)

		if err != nil {
			m.Logger().Errorf("error trying to logout from vSphere: %v", err)
		}
	}()

	c := client.Client

	// Get custom fields (attributes) names if get_custom_fields is true.
	customFieldsMap := make(map[int32]string)
	if m.GetCustomFields {
		var err error
		customFieldsMap, err = setCustomFieldsMap(ctx, c)
		if err != nil {
			return fmt.Errorf("virtualmachine: error in setCustomFieldsMap: %w", err)
		}
	}

	// Create view of VirtualMachine objects
	mgr := view.NewManager(c)

	v, err := mgr.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return fmt.Errorf("virtualmachine: error in CreateContainerView: %w", err)
	}

	defer func() {
		if err := v.Destroy(ctx); err != nil {
			m.Logger().Debugf("Error destroying view from vsphere %v", err)
		}
	}()

	// Retrieve summary property for all machines
	var vmt []mo.VirtualMachine
	err = v.Retrieve(ctx, []string{"VirtualMachine"}, []string{"summary", "datastore", "triggeredAlarmState", "snapshot"}, &vmt)
	if err != nil {
		return fmt.Errorf("virtualmachine: error in Retrieve: %w", err)
	}

	pc := property.DefaultCollector(c)
	for _, vm := range vmt {
		var hostID, hostName string
		var networkNames, datastoreNames []string
		var customFields mapstr.M
		var snapshots []VMSnapshotData

		if host := vm.Summary.Runtime.Host; host != nil {
			hostID = host.Value
			hostSystem, err := getHostSystem(ctx, c, host.Reference())
			if err == nil {
				hostName = hostSystem.Summary.Config.Name
			} else {
				m.Logger().Debug(err.Error())
			}
		} else {
			m.Logger().Debug("'Host', 'Runtime' or 'Summary' data not found. This is either a parsing error " +
				"from vsphere library, an error trying to reach host/guest or incomplete information returned " +
				"from host/guest")
		}

		// Retrieve custom fields if enabled
		if m.GetCustomFields && vm.Summary.CustomValue != nil {
			customFields = getCustomFields(vm.Summary.CustomValue, customFieldsMap)
		}
		if len(customFields) <= 0 {
			m.Logger().Debug("custom fields not activated or custom values not found/parse in Summary data. This " +
				"is either a parsing error from vsphere library, an error trying to reach host/guest or incomplete " +
				"information returned from host/guest")
		}
		// Retrieve network names
		if vm.Summary.Vm != nil {
			networkNames, err = getNetworkNames(ctx, c, vm.Summary.Vm.Reference())
			if err != nil {
				m.Logger().Debug(err.Error())
			}
		}

		// Retrieve the datastore names associated with the Virtualmachine
		for _, datastoreRef := range vm.Datastore {
			var ds mo.Datastore
			err = pc.RetrieveOne(ctx, datastoreRef, []string{"name"}, &ds)
			if err == nil {
				datastoreNames = append(datastoreNames, ds.Name)
			} else {
				m.Logger().Debug("error retrieving datastore name for VM %s: %v", vm.Summary.Config.Name, err)
			}
		}

		if vm.Snapshot != nil {
			snapshots = fetchSnapshots(vm.Snapshot.RootSnapshotList)
		}

		triggeredAlarm, err := getTriggeredAlarm(ctx, pc, vm.TriggeredAlarmState)
		if err != nil {
			m.Logger().Errorf("Failed to retrieve alerts from VM %s: %w", vm.Name, err)
		}

		data := VMData{
			VM:              vm,
			HostID:          hostID,
			HostName:        hostName,
			NetworkNames:    networkNames,
			DatastoreNames:  datastoreNames,
			CustomFields:    customFields,
			Snapshots:       snapshots,
			triggeredAlarms: triggeredAlarm,
		}

		reporter.Event(mb.Event{
			MetricSetFields: m.mapEvent(data),
		})
	}

	return nil
}

func getCustomFields(customFields []types.BaseCustomFieldValue, customFieldsMap map[int32]string) mapstr.M {
	outputFields := mapstr.M{}
	for _, v := range customFields {
		customFieldString, customFieldCastOk := v.(*types.CustomFieldStringValue)
		key, ok := customFieldsMap[v.GetCustomFieldValue().Key]
		if customFieldCastOk && ok {
			// If key has '.', is replaced with '_' to be compatible with ES2.x.
			fmtKey := strings.ReplaceAll(key, ".", "_")
			outputFields.Put(fmtKey, customFieldString.Value)
		}
	}

	return outputFields
}

func getNetworkNames(ctx context.Context, c *vim25.Client, ref types.ManagedObjectReference) ([]string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pc := property.DefaultCollector(c)

	var vm mo.VirtualMachine
	err := pc.RetrieveOne(ctx, ref, []string{"network"}, &vm)
	if err != nil {
		return nil, fmt.Errorf("error retrieving virtual machine information: %w", err)
	}

	if len(vm.Network) == 0 {
		return nil, errors.New("no networks found")
	}

	var nets []mo.Network
	if err := pc.Retrieve(ctx, vm.Network, []string{"name"}, &nets); err != nil {
		return nil, fmt.Errorf("error retrieving network from virtual machine: %w", err)
	}
	outputNetworkNames := make([]string, 0, len(nets))
	for _, net := range nets {
		name := strings.ReplaceAll(net.Name, ".", "_")
		outputNetworkNames = append(outputNetworkNames, name)
	}

	return outputNetworkNames, nil
}

func setCustomFieldsMap(ctx context.Context, client *vim25.Client) (map[int32]string, error) {
	customFieldsMap := make(map[int32]string)

	customFieldsManager, err := object.GetCustomFieldsManager(client)

	if err != nil {
		return nil, fmt.Errorf("failed to get custom fields manager: %w", err)
	}
	field, err := customFieldsManager.Field(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get custom fields: %w", err)
	}

	for _, def := range field {
		customFieldsMap[def.Key] = def.Name
	}

	return customFieldsMap, nil
}

func getHostSystem(ctx context.Context, c *vim25.Client, ref types.ManagedObjectReference) (*mo.HostSystem, error) {
	pc := property.DefaultCollector(c)

	var hs mo.HostSystem
	err := pc.RetrieveOne(ctx, ref, []string{"summary"}, &hs)
	if err != nil {
		return nil, fmt.Errorf("error retrieving host information: %w", err)
	}
	return &hs, nil
}

func fetchSnapshots(snapshotTree []types.VirtualMachineSnapshotTree) []VMSnapshotData {
	snapshots := make([]VMSnapshotData, 0, len(snapshotTree))
	for _, snapshot := range snapshotTree {
		snapshots = append(snapshots, VMSnapshotData{
			ID:          snapshot.Id,
			Name:        snapshot.Name,
			Description: snapshot.Description,
			CreateTime:  common.Time(snapshot.CreateTime),
			State:       snapshot.State,
		})

		// Recursively add child snapshots
		if len(snapshot.ChildSnapshotList) > 0 {
			snapshots = append(snapshots, fetchSnapshots(snapshot.ChildSnapshotList)...)
		}
	}
	return snapshots
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
