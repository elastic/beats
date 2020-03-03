/*
Copyright (c) 2017 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package simulator

import (
	"time"

	"github.com/google/uuid"
	"github.com/vmware/govmomi/simulator/esx"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

type HostSystem struct {
	mo.HostSystem
}

func NewHostSystem(host mo.HostSystem) *HostSystem {
	now := time.Now()

	host.Name = host.Summary.Config.Name
	host.Summary.Runtime = &host.Runtime
	host.Summary.Runtime.BootTime = &now

	hw := *host.Summary.Hardware // shallow copy
	hw.Uuid = uuid.New().String()
	host.Summary.Hardware = &hw

	info := *esx.HostHardwareInfo
	info.SystemInfo.Uuid = hw.Uuid
	host.Hardware = &info

	hs := &HostSystem{
		HostSystem: host,
	}

	config := []struct {
		ref **types.ManagedObjectReference
		obj mo.Reference
	}{
		{&hs.ConfigManager.DatastoreSystem, &HostDatastoreSystem{Host: &hs.HostSystem}},
		{&hs.ConfigManager.NetworkSystem, NewHostNetworkSystem(&hs.HostSystem)},
		{&hs.ConfigManager.AdvancedOption, NewOptionManager(nil, esx.Setting)},
		{&hs.ConfigManager.FirewallSystem, NewHostFirewallSystem(&hs.HostSystem)},
	}

	for _, c := range config {
		ref := Map.Put(c.obj).Reference()

		*c.ref = &ref
	}

	return hs
}

func hostParent(host *mo.HostSystem) *mo.ComputeResource {
	switch parent := Map.Get(*host.Parent).(type) {
	case *mo.ComputeResource:
		return parent
	case *ClusterComputeResource:
		return &parent.ComputeResource
	default:
		return nil
	}
}

// CreateDefaultESX creates a standalone ESX
// Adds objects of type: Datacenter, Network, ComputeResource, ResourcePool and HostSystem
func CreateDefaultESX(f *Folder) {
	dc := &esx.Datacenter
	f.putChild(dc)
	createDatacenterFolders(dc, false)

	host := NewHostSystem(esx.HostSystem)

	cr := &mo.ComputeResource{}
	cr.Self = *host.Parent
	cr.Name = host.Name
	cr.Host = append(cr.Host, host.Reference())
	Map.PutEntity(cr, host)

	pool := NewResourcePool()
	cr.ResourcePool = &pool.Self
	Map.PutEntity(cr, pool)
	pool.Owner = cr.Self

	Map.Get(dc.HostFolder).(*Folder).putChild(cr)
}

// CreateStandaloneHost uses esx.HostSystem as a template, applying the given spec
// and creating the ComputeResource parent and ResourcePool sibling.
func CreateStandaloneHost(f *Folder, spec types.HostConnectSpec) (*HostSystem, types.BaseMethodFault) {
	if spec.HostName == "" {
		return nil, &types.NoHost{}
	}

	pool := NewResourcePool()
	host := NewHostSystem(esx.HostSystem)

	host.Summary.Config.Name = spec.HostName
	host.Name = host.Summary.Config.Name
	host.Runtime.ConnectionState = types.HostSystemConnectionStateDisconnected

	cr := &mo.ComputeResource{}

	Map.PutEntity(cr, Map.NewEntity(host))

	Map.PutEntity(cr, Map.NewEntity(pool))

	cr.Name = host.Name
	cr.Host = append(cr.Host, host.Reference())
	cr.ResourcePool = &pool.Self

	f.putChild(cr)
	pool.Owner = cr.Self

	return host, nil
}
