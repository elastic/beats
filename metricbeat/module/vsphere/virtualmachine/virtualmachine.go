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
	"log"
	"strings"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/vsphere"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/performance"
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
	MetricLevel     int
}

// New creates a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := vsphere.NewMetricSet(base)
	if err != nil {
		return nil, err
	}

	config := struct {
		GetCustomFields bool `config:"get_custom_fields"`
		MetricLevel     int  `config:"metric_level"`
	}{
		GetCustomFields: false,
		MetricLevel:     1,
	}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
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
		return fmt.Errorf("error in NewClient: %w", err)
	}

	defer func() {
		if err := client.Logout(ctx); err != nil {
			m.Logger().Debug(fmt.Errorf("error trying to logout from vshphere: %w", err))
		}
	}()

	c := client.Client

	// Get custom fields (attributes) names if get_custom_fields is true.
	customFieldsMap := make(map[int32]string)
	if m.GetCustomFields {
		var err error
		customFieldsMap, err = setCustomFieldsMap(ctx, c)
		if err != nil {
			return fmt.Errorf("error in setCustomFieldsMap: %w", err)
		}
	}

	// Create view of VirtualMachine objects
	mgr := view.NewManager(c)

	v, err := mgr.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return fmt.Errorf("error in CreateContainerView: %w", err)
	}

	defer func() {
		if err := v.Destroy(ctx); err != nil {
			m.Logger().Debug(fmt.Errorf("error trying to destroy view from vshphere: %w", err))
		}
	}()

	r, err := mgr.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"ResourcePool"}, true)
	if err != nil {
		return fmt.Errorf("error in CreateContainerView: %w", err)
	}

	defer func() {
		if err := r.Destroy(ctx); err != nil {
			m.Logger().Debug(fmt.Errorf("error trying to destroy view from vshphere: %w", err))
		}
	}()

	// Retrieve summary property for all machines
	var vmt []mo.VirtualMachine
	err = v.Retrieve(ctx, []string{"VirtualMachine"}, []string{"summary", "resourcePool"}, &vmt)
	if err != nil {
		return fmt.Errorf("error in Retrieve: %w", err)
	}

	// Create a map to easily look up VM names by ID
	vmMap := make(map[string]mo.VirtualMachine)
	for _, vm := range vmt {
		vmMap[vm.Reference().Value] = vm
	}

	var vmsToQuery []types.ManagedObjectReference

	for _, vm := range vmt {
		vmsToQuery = append(vmsToQuery, vm.Reference())
	}

	// Retrieve all resource pools in the cluster for later mapping
	var rps []mo.ResourcePool
	err = r.Retrieve(ctx, []string{"ResourcePool"}, []string{"summary"}, &rps)
	if err != nil {
		return fmt.Errorf("error in Retrieve: %w", err)
	}

	// Create a map to easily look up resource pools by name
	rpMap := make(map[string]mo.ResourcePool)
	for _, rp := range rps {
		rpMap[rp.Name] = rp
	}

	// Create a Performance Manager instance
	perfManager := performance.NewManager(c)

	// gets a counterInfo object that describes all available counters for virtualMachines
	counterInfo, err := perfManager.CounterInfo(ctx)
	if err != nil {
		log.Fatalf("Error retrieving counter info: %v", err)
	}

	// Create a map to easily look up counter names by ID
	counterMap := make(map[int32]string)
	for _, counter := range counterInfo {
		counterMap[counter.Key] = counter.Name()
	}

	querySpec := types.PerfQuerySpec{
		IntervalId: 20, // this likely needs to be made into a variable to support other-than-realtime metrics but it's not clear how to define that in the config and map to this and Level correctly
		MaxSample:  1,
	}

	// Create a slice to hold our query specifications for each VM
	var metricsToQuery []string

	// Iterate over the counterInfo slice
	for _, counter := range counterInfo {
		// Check if the counter is collected at a 20-second interval
		if counter.Level == int32(m.MetricLevel) {
			// Add the counter to the metricsToQuery slice
			metricsToQuery = append(metricsToQuery, counter.Name())

		}
	}

	// Perform the query for all VMs at once
	metricBase, err := perfManager.SampleByName(ctx, querySpec, metricsToQuery, vmsToQuery)
	if err != nil {
		fmt.Printf("Error retrieving metrics: %s\n", err)
		return nil // this probably needs to be a different error
	}

	// assign metrics to the event output
	for _, base := range metricBase {
		if metric, ok := base.(*types.PerfEntityMetric); ok {
			virtualMachine := vmMap[metric.Entity.Value]
			// resourcePool := rpMap[virtualMachine.ResourcePool.Value] // not sure why but the Summary field isn't allowing me to get Name even though the docs say it's there
			event := mapstr.M{
				"name": virtualMachine.Summary.Config.Name,
				"os":   virtualMachine.Summary.Config.GuestFullName,
				"uuid": virtualMachine.Summary.Config.InstanceUuid,
				"id":   virtualMachine.Reference().Value,
				"resource_pool": mapstr.M{
					"id": virtualMachine.ResourcePool.Value,
				},
				"cpu": mapstr.M{
					"reserved": virtualMachine.Summary.Config.CpuReservation,
					"cores":    virtualMachine.Summary.Config.NumCpu,
				},
				"disks": mapstr.M{
					"count": virtualMachine.Summary.Config.NumVirtualDisks,
				},
				"storage": mapstr.M{
					"committed":  virtualMachine.Summary.Storage.Committed,
					"uncommited": virtualMachine.Summary.Storage.Uncommitted,
					"total":      virtualMachine.Summary.Storage.Committed + virtualMachine.Summary.Storage.Uncommitted,
				},
				"heartbeat_status": virtualMachine.GuestHeartbeatStatus,
				"connection_state": virtualMachine.Summary.Runtime.ConnectionState,
				"memory": mapstr.M{
					"overhead":   virtualMachine.Summary.Runtime.MemoryOverhead,
					"total_size": virtualMachine.Summary.Config.MemorySizeMB,
					"reserved":   virtualMachine.Summary.Config.MemoryReservation,
				},
				"power_state":                   virtualMachine.Summary.Runtime.PowerState,
				"snapshot_consolidation_needed": virtualMachine.Summary.Runtime.ConsolidationNeeded,
				"vmx_path":                      virtualMachine.Summary.Config.VmPathName,
			}
			for _, value := range metric.Value {
				switch series := value.(type) {
				case *types.PerfMetricIntSeries:
					counter := counterMap[series.Id.CounterId]
					event.Put(counter, series.Value)
				case *types.PerfMetricSeriesCSV:
					counter := counterMap[series.Id.CounterId]
					event.Put(counter, series.Value)
				default:
					m.Logger().Debug("Metric is of an unknown type, skipping")
				}
			}

			// Get host information for VM
			if host := virtualMachine.Summary.Runtime.Host; host != nil {
				event["host"] = mapstr.M{
					"id": host.Value,
				}
				hostSystem, err := getHostSystem(ctx, c, host.Reference())
				if err == nil {
					event.Put("host.hostname", hostSystem.Summary.Config.Name)
				} else {
					m.Logger().Debug(err.Error())
				}
			} else {
				m.Logger().Debug("'Host', 'Runtime' or 'Summary' data not found. This is either a parsing error " +
					"from vsphere library, an error trying to reach host/guest or incomplete information returned " +
					"from host/guest")
			}

			// Get custom fields (attributes) values if get_custom_fields is true.
			if m.GetCustomFields && virtualMachine.Summary.CustomValue != nil {
				customFields := getCustomFields(virtualMachine.Summary.CustomValue, customFieldsMap)

				if len(customFields) > 0 {
					event["custom_fields"] = customFields
				}
			} else {
				m.Logger().Debug("custom fields not activated or custom values not found/parse in Summary data. This " +
					"is either a parsing error from vsphere library, an error trying to reach host/guest or incomplete " +
					"information returned from host/guest")
			}

			if virtualMachine.Summary.Vm != nil {
				networkNames, err := getNetworkNames(ctx, c, virtualMachine.Summary.Vm.Reference())
				if err != nil {
					m.Logger().Debug(err.Error())
				} else {
					if len(networkNames) > 0 {
						event["network_names"] = networkNames
					}
				}
			}

			if virtualMachine.Datastore != nil {
				var datastoreIds []string
				for _, datastore := range virtualMachine.Datastore {
					datastoreIds = append(datastoreIds, datastore.Value)
				}
				event["datastore.id"] = datastoreIds
			}

			reporter.Event(mb.Event{
				MetricSetFields: event,
			})
		}
	}

	return nil
}

func getCustomFields(customFields []types.BaseCustomFieldValue, customFieldsMap map[int32]string) mapstr.M {
	outputFields := mapstr.M{}
	for _, v := range customFields {
		customFieldString := v.(*types.CustomFieldStringValue)
		key, ok := customFieldsMap[v.GetCustomFieldValue().Key]
		if ok {
			// If key has '.', is replaced with '_' to be compatible with ES2.x.
			fmtKey := strings.Replace(key, ".", "_", -1)
			outputFields.Put(fmtKey, customFieldString.Value)
		}
	}

	return outputFields
}

func getNetworkNames(ctx context.Context, c *vim25.Client, ref types.ManagedObjectReference) ([]string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var outputNetworkNames []string

	pc := property.DefaultCollector(c)

	var vm mo.VirtualMachine
	err := pc.RetrieveOne(ctx, ref, []string{"network"}, &vm)
	if err != nil {
		return nil, fmt.Errorf("error retrieving virtual machine information: %v", err)
	}

	if len(vm.Network) == 0 {
		return nil, errors.New("no networks found")
	}

	var networkRefs []types.ManagedObjectReference
	for _, obj := range vm.Network {
		if obj.Type == "Network" {
			networkRefs = append(networkRefs, obj)
		}
	}

	// If only "Distributed port group" was found, for example.
	if len(networkRefs) == 0 {
		return nil, errors.New("no networks found")
	}

	var nets []mo.Network
	err = pc.Retrieve(ctx, networkRefs, []string{"name"}, &nets)
	if err != nil {
		return nil, fmt.Errorf("error retrieving network from virtual machine: %v", err)
	}

	for _, net := range nets {
		name := strings.Replace(net.Name, ".", "_", -1)
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
		return nil, fmt.Errorf("error retrieving host information: %v", err)
	}
	return &hs, nil
}
