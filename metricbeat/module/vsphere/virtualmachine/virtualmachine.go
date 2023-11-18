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
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/exp/slices"
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
	MaxQuerySize    int
	ResourcePools   []string
	DataCenters     []string
}

// New creates a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := vsphere.NewMetricSet(base)
	if err != nil {
		return nil, err
	}

	config := struct {
		GetCustomFields bool     `config:"get_custom_fields"`
		MetricLevel     int      `config:"metric_level"`
		MaxQuerySize    int      `config:"max_query_size"`
		ResourcePools   []string `config:"resource_pools"`
		DataCenters     []string `config:"data_centers"`
	}{
		GetCustomFields: false,
		MetricLevel:     1,
		MaxQuerySize:    256,
		ResourcePools:   nil,
		DataCenters:     nil,
	}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	return &MetricSet{
		MetricSet:       ms,
		GetCustomFields: config.GetCustomFields,
		MetricLevel:     config.MetricLevel,
		MaxQuerySize:    config.MaxQuerySize,
		ResourcePools:   config.ResourcePools,
		DataCenters:     config.DataCenters,
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

	rpMap, err := m.getResourcePoolMap(ctx, c, mgr)
	if err != nil {
		return fmt.Errorf("error getting Resource Pool Map: %w", err)
	}
	var rpList = make([]string, 0, len(rpMap))
	for _, rp := range rpMap {
		rpList = append(rpList, rp.ref.Value)
	}

	// Create a Performance Manager instance
	perfManager := performance.NewManager(c)

	// gets a counterInfo object that describes all available counters for virtualMachines
	counterInfo, err := perfManager.CounterInfo(ctx)
	if err != nil {
		log.Fatalf("Error retrieving counter info: %v", err)
	}

	// Create a map to easily look up counter names by ID
	counterMap := make(map[int32]string, len(counterInfo))
	for _, counter := range counterInfo {
		counterMap[counter.Key] = counter.Name()
	}

	querySpec := types.PerfQuerySpec{
		IntervalId: 20, // NOTE: This likely needs to be made into a variable to support other-than-realtime metrics but it's not clear how to define that in the config and map to this and Level correctly
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

	vmList, err := m.getVirtualMachineList(ctx, c, mgr, rpList)
	if err != nil {
		return fmt.Errorf("error getting Virtual Machine List: %w", err)
	}

	vmChunks := m.getVmChunks(vmList)
	vmMap := getVmMap(vmList)

	for _, chunk := range vmChunks {
		// Perform the query for all VMs at once
		metricBase, err := perfManager.SampleByName(ctx, querySpec, metricsToQuery, chunk)
		if err != nil {
			return fmt.Errorf("Error retrieving metrics: %s\n", err)
		}

		// Assign metrics to the event output
		for _, base := range metricBase {
			metric, ok := base.(*types.PerfEntityMetric)
			if !ok {
				continue
			}

			// Build initial event object with static values
			virtualMachine := vmMap[metric.Entity.Value]
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

			// add mapped values
			resourcePool := rpMap[virtualMachine.ResourcePool.Value]
			event.Put("resource_pool.name", resourcePool.Name)
			event.Put("resource_pool.path", resourcePool.InventoryPath)

			for _, value := range metric.Value {
				switch series := value.(type) {
				case *types.PerfMetricIntSeries:
					counter := counterMap[series.Id.CounterId]
					event.Put(counter, series.Value)
				case *types.PerfMetricSeriesCSV:
					counter := counterMap[series.Id.CounterId]
					event.Put(counter, series.Value)
				default:
					m.Logger().Debug("metric is of an unknown type, skipping")
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
				var datastoreIds = make([]string, 0, len(virtualMachine.Datastore))
				for _, datastore := range virtualMachine.Datastore {
					datastoreIds = append(datastoreIds, datastore.Value)
				}
				event["datastore.id"] = datastoreIds
			}

			reporter.Event(mb.Event{MetricSetFields: event})
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

func splitIntoChunks(slice []types.ManagedObjectReference, n int) [][]types.ManagedObjectReference {
	var chunks [][]types.ManagedObjectReference

	for i := 0; i < len(slice); i += n {
		end := i + n

		// Check if the end index is beyond slice bounds
		if end > len(slice) {
			end = len(slice)
		}

		chunks = append(chunks, slice[i:end])
	}

	return chunks
}

func getVmFilter(pools []types.ManagedObjectReference) property.Filter {
	return property.Filter{"resourcePool": pools}
}

func (m *MetricSet) getVirtualMachineList(ctx context.Context, c *vim25.Client, mgr *view.Manager, includedPools []string) ([]mo.VirtualMachine, error) {
	v, err := mgr.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return nil, fmt.Errorf("error in CreateContainerView: %w", err)
	}

	defer func() {
		if err := v.Destroy(ctx); err != nil {
			m.Logger().Debugf("error trying to destroy view from vshphere: %s", err)
		}
	}()

	var vmt []mo.VirtualMachine

	err = v.Retrieve(ctx, []string{"VirtualMachine"}, []string{"summary", "config", "resourcePool"}, &vmt)
	if err != nil {
		return nil, fmt.Errorf("error in Retrieve: %w", err)
	}

	// Skip filtering the list if no pools are specified
	if m.ResourcePools == nil && m.DataCenters == nil {
		return vmt, nil
	}

	var vms []mo.VirtualMachine
	for _, vm := range vmt {
		if vm.ResourcePool != nil {
			m.Logger().Debugf("vm %s has rpool %s", vm.Summary.Config.Name, vm.ResourcePool.Value)
			if slices.Contains(includedPools, vm.ResourcePool.Value) {
				m.Logger().Debugf("adding vm %s", vm.Summary.Config.Name)
				vms = append(vms, vm)
			}
		} else {
			m.Logger().Warnf("failed to get resource pool info for: %s as there is no rpool reference, skipping", vm.Summary.Config.Name)
		}
	}

	return vms, nil

}

func (m *MetricSet) getVmChunks(vms []mo.VirtualMachine) [][]types.ManagedObjectReference {
	var vmsToQuery = make([]types.ManagedObjectReference, 0, len(vms))

	for _, vm := range vms {
		vmsToQuery = append(vmsToQuery, vm.Reference())
	}
	vmChunks := splitIntoChunks(vmsToQuery, m.MaxQuerySize)
	return vmChunks
}

func getVmMap(vms []mo.VirtualMachine) map[string]mo.VirtualMachine {
	vmMap := make(map[string]mo.VirtualMachine)
	for _, vm := range vms {
		vmMap[vm.Reference().Value] = vm
	}
	return vmMap
}

type resourcePoolInfo struct {
	InventoryPath string
	Name          string
	ref           types.ManagedObjectReference
}

func (m *MetricSet) getResourcePoolMap(ctx context.Context, c *vim25.Client, mgr *view.Manager) (map[string]resourcePoolInfo, error) {
	// Create a single ContainerView for ResourcePool types
	r, err := mgr.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"ResourcePool"}, true)
	if err != nil {
		return nil, err
	}
	defer r.Destroy(ctx) // Simplified error handling, as the error is not critical

	// Retrieve all resource pools in the cluster for later mapping
	var rps []mo.ResourcePool
	err = r.Retrieve(ctx, []string{"ResourcePool"}, []string{"summary"}, &rps)
	if err != nil {
		return nil, err
	}

	// Create a map to easily look up resource pools by reference
	rpMap := make(map[string]resourcePoolInfo, len(rps))
	for _, rp := range rps {
		summary := rp.Summary.GetResourcePoolSummary()
		rpInfo := resourcePoolInfo{
			InventoryPath: "", // Placeholder, as we don't have the InventoryPath directly
			Name:          summary.Name,
		}
		rpMap[rp.Reference().Value] = rpInfo
	}

	// Now, we need to fill in the InventoryPath for each resource pool
	finder := find.NewFinder(c, false)
	dcList, err := finder.DatacenterList(ctx, "*")
	if err != nil {
		return nil, err
	}
	for _, dc := range dcList {
		if m.DataCenters != nil && !slices.Contains(m.DataCenters, dc.Name()) {
			continue
		}
		finder.SetDatacenter(dc)
		pools, err := finder.ResourcePoolList(ctx, "*")
		if err != nil {
			return nil, fmt.Errorf("unable to find resource pools for %s due to %w", dc.Name(), err)
		}
		for _, pool := range pools {
			if m.ResourcePools != nil && !slices.Contains(m.ResourcePools, pool.InventoryPath) {
				m.Logger().Debugf("skipping resource pool %s as it is not in the list of included resource pools", pool.InventoryPath)
				continue
			}
			if rpInfo, exists := rpMap[pool.Reference().Value]; exists {
				rpInfo.InventoryPath = pool.InventoryPath
				rpInfo.ref = pool.Reference()
				rpMap[pool.Reference().Value] = rpInfo // Update the map with the InventoryPath
			}
		}
	}

	m.Logger().Info("retrieved resource pool list")
	return rpMap, nil
}

// getVMInventoryPath is currently unused since it's a very expensive call, but if we could cache it it would be nice to have
func getVMInventoryPath(ctx context.Context, c *vim25.Client, vm mo.VirtualMachine) (string, error) {
	// Create a property collector
	pc := property.DefaultCollector(c)

	// Traverse up the inventory tree to construct the full path
	var pathElements []string
	ref := vm.Reference()
	for ref.Type != "Datacenter" {
		var entity mo.ManagedEntity
		err := pc.RetrieveOne(ctx, ref, []string{"name", "parent"}, &entity)
		if err != nil {
			return "", err
		}
		pathElements = append([]string{entity.Name}, pathElements...)
		if entity.Parent == nil {
			break
		}
		ref = *entity.Parent
	}

	// Join the path elements
	inventoryPath := "/" + strings.Join(pathElements, "/")

	return inventoryPath, nil
}
