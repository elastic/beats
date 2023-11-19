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
	RpMap           map[string]resourcePoolInfo
	HostMap         map[string]hostInfo
	DsMap           map[string]DatastoreInfo
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
		RpMap:           make(map[string]resourcePoolInfo),
		HostMap:         make(map[string]hostInfo),
		DsMap:           make(map[string]DatastoreInfo),
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

	// if perform initial population of the host, resource pool, network, and datastore maps
	if len(m.RpMap) == 0 {
		m.RpMap, err = m.getResourcePoolMap(ctx, c, mgr)
		if err != nil {
			return fmt.Errorf("error getting Resource Pool Map: %w", err)
		}
	}
	if len(m.HostMap) == 0 {
		m.HostMap, err = m.getHostMap(ctx, c, mgr)
		if err != nil {
			return fmt.Errorf("error getting Host Map: %w", err)
		}
	}
	if len(m.DsMap) == 0 {
		m.DsMap, err = m.getDatastoreMap(ctx, c, mgr)
		if err != nil {
			return fmt.Errorf("error getting Datastore Map: %w", err)
		}
	}

	// create bool variables that can be set to true if refreshes are needed
	var rpRefreshRequired bool
	var hostRefreshRequired bool
	var dsRefreshRequired bool

	var rpList = make([]string, 0, len(m.RpMap))
	for _, rp := range m.RpMap {
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
				"name":             virtualMachine.Summary.Config.Name,
				"os":               virtualMachine.Summary.Config.GuestFullName,
				"os_family":        virtualMachine.Guest.GuestFamily,
				"guest_state":      virtualMachine.Guest.GuestState,
				"hardware_version": virtualMachine.Guest.HwVersion,
				"vmtools_version":  virtualMachine.Guest.ToolsVersion,
				"uuid":             virtualMachine.Summary.Config.InstanceUuid,
				"id":               virtualMachine.Reference().Value,
				"primary_ip":       virtualMachine.Guest.IpAddress,
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

			// Add resource pool name and path if available, if not, set the rpRefreshRequired var to true
			resourcePool, ok := m.RpMap[virtualMachine.ResourcePool.Value]
			if !ok {
				rpRefreshRequired = true
				m.Logger().Debugf("resource pool with id %s not found, will refresh pool information at end of current run", virtualMachine.ResourcePool.Value)
			} else {
				event.Put("resource_pool.name", resourcePool.Name)
				event.Put("resource_pool.path", resourcePool.InventoryPath)
			}

			if virtualMachine.Datastore != nil {
				var datastores []DatastoreInfo
				for _, ds := range virtualMachine.Datastore {
					datastore, ok := m.DsMap[ds.Value]
					if !ok {
						dsRefreshRequired = true
						m.Logger().Debugf("datastore with id %s not found, will refresh datastore information at the end of the run")
					} else {
						datastores = append(datastores, datastore)
						event.Put("datastores", datastores)
					}
				}
			}

			// Get host information for VM
			vmhost, ok := m.HostMap[virtualMachine.Summary.Runtime.Host.Value]
			if !ok {
				hostRefreshRequired = true
			} else {
				event.Put("host.id", vmhost.Ref.Value)
				event.Put("host.hostname", vmhost.Hostname)
				event.Put("host.version", vmhost.Version)
			}

			var networks []NetworkInfo
			for _, netInfo := range virtualMachine.Guest.Net {
				network := NetworkInfo{
					Network:    netInfo.Network,
					MacAddress: netInfo.MacAddress,
					Connected:  netInfo.Connected,
					// DNSAddresses: netInfo.DnsConfig.IpAddress,
				}
				if netInfo.DnsConfig != nil {
					network.DNSAddresses = netInfo.DnsConfig.IpAddress
				}
				if netInfo.IpConfig != nil {
					for _, ipConfig := range netInfo.IpConfig.IpAddress {
						network.IPConfig = append(network.IPConfig, IPConfig{
							IPAddress:    ipConfig.IpAddress,
							PrefixLength: int(ipConfig.PrefixLength),
						})
					}
				}
				networks = append(networks, network)
			}
			event.Put("networks", networks)

			var disks []DiskInfo
			for _, diskInfo := range virtualMachine.Guest.Disk {
				disk := DiskInfo{
					Capacity:  diskInfo.Capacity,
					Path:      diskInfo.DiskPath,
					FreeSpace: diskInfo.FreeSpace,
				}
				disks = append(disks, disk)
			}
			event.Put("disks", disks)

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

			reporter.Event(mb.Event{MetricSetFields: event})
		}
	}

	if rpRefreshRequired {
		m.RpMap, err = m.getResourcePoolMap(ctx, c, mgr)
		if err != nil {
			return fmt.Errorf("error getting Resource Pool Map: %w", err)
		}
	}

	if hostRefreshRequired {
		m.HostMap, err = m.getHostMap(ctx, c, mgr)
		if err != nil {
			return fmt.Errorf("error getting Host Map: %w", err)
		}
	}

	if dsRefreshRequired {
		m.DsMap, err = m.getDatastoreMap(ctx, c, mgr)
		if err != nil {
			return fmt.Errorf("error getting datastore map %w", err)
		}
	}

	return nil
}

// Defines the stucts that map IP information together
type NetworkInfo struct {
	Network      string     `json:"network"`
	MacAddress   string     `json:"mac_address"`
	Connected    bool       `json:"connected"`
	IPConfig     []IPConfig `json:"ip_config"`
	DNSAddresses []string   `json:"dns_addresses,omitempty"` // Optional field
}

type IPConfig struct {
	IPAddress    string `json:"ip_address"`
	PrefixLength int    `json:"prefix_length"`
}

type DNSConfig struct {
	DNSAddresses []string `json:"dns_addresses"`
	DNSDomain    string   `json:"dns_domain,omitempty"` // Optional field
}

type DiskInfo struct {
	Capacity  int64  `json:"capacity"`
	Path      string `json:"path"`
	FreeSpace int64  `json:"free_space"`
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

	err = v.Retrieve(ctx, []string{"VirtualMachine"}, []string{"summary", "config", "resourcePool", "guest"}, &vmt)
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

type hostInfo struct {
	Ref      types.ManagedObjectReference
	Hostname string
	Version  string
}

func (m *MetricSet) getHostMap(ctx context.Context, c *vim25.Client, mgr *view.Manager) (map[string]hostInfo, error) {
	// Create a single ContainerView for Host types
	r, err := mgr.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"HostSystem"}, true)
	if err != nil {
		return nil, err
	}
	defer r.Destroy(ctx) // Simplified error handling, as the error is not critical

	var hosts []mo.HostSystem
	err = r.Retrieve(ctx, []string{"HostSystem"}, []string{"summary"}, &hosts)
	if err != nil {
		return nil, err
	}

	hostMap := make(map[string]hostInfo, len(hosts))
	for _, host := range hosts {
		summary := host.Summary
		info := hostInfo{
			Ref:      host.Reference(),
			Hostname: summary.Config.Name,
			Version:  summary.Config.Product.Version,
		}
		hostMap[host.Reference().Value] = info
	}

	m.Logger().Info("retrieved host list")
	return hostMap, nil
}

type DatastoreInfo struct {
	Name string `json:"name"`
	Id   string `json:"id"`
	Type string `json:"type"`
}

func (m *MetricSet) getDatastoreMap(ctx context.Context, c *vim25.Client, mgr *view.Manager) (map[string]DatastoreInfo, error) {
	r, err := mgr.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"Datastore"}, true)
	if err != nil {
		return nil, err
	}
	defer r.Destroy(ctx) // Simplified error handling, as the error is not critical

	var datastores []mo.Datastore
	err = r.Retrieve(ctx, []string{"Datastore"}, []string{"summary"}, &datastores)
	if err != nil {
		return nil, err
	}

	datastoreMap := make(map[string]DatastoreInfo, len(datastores))
	for _, datastore := range datastores {
		info := DatastoreInfo{
			Id:   datastore.Reference().Value,
			Name: datastore.Summary.Name,
			Type: datastore.Summary.Type,
		}
		datastoreMap[datastore.Reference().Value] = info
	}
	m.Logger().Info("retrieved datastore list")
	return datastoreMap, nil
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
