// Copyright 2016 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package simulator

import (
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
	"github.com/vmware/vic/pkg/vsphere/simulator/esx"
)

type ClusterComputeResource struct {
	mo.ClusterComputeResource
}

type addHostTask struct {
	*ClusterComputeResource

	req *types.AddHost_Task
}

func (add *addHostTask) Run(task *Task) (types.AnyType, types.BaseMethodFault) {
	spec := add.req.Spec

	if spec.HostName == "" {
		return nil, &types.NoHost{}
	}

	host := NewHostSystem(esx.HostSystem)
	host.Summary.Config.Name = spec.HostName
	host.Name = host.Summary.Config.Name
	host.Runtime.ConnectionState = types.HostSystemConnectionStateDisconnected

	cr := add.ClusterComputeResource
	Map.PutEntity(cr, Map.NewEntity(host))

	cr.Host = append(cr.Host, host.Reference())

	if add.req.AsConnected {
		host.Runtime.ConnectionState = types.HostSystemConnectionStateConnected
	}

	return host.Reference(), nil
}

func (c *ClusterComputeResource) AddHostTask(add *types.AddHost_Task) soap.HasFault {
	r := &methods.AddHost_TaskBody{}

	task := NewTask(&addHostTask{c, add})

	r.Res = &types.AddHost_TaskResponse{
		Returnval: task.Self,
	}

	task.Run()

	return r
}

func CreateClusterComputeResource(f *Folder, name string, spec types.ClusterConfigSpecEx) (*ClusterComputeResource, types.BaseMethodFault) {
	if e := Map.FindByName(name, f.ChildEntity); e != nil {
		return nil, &types.DuplicateName{
			Name:   e.Entity().Name,
			Object: e.Reference(),
		}
	}

	cluster := &ClusterComputeResource{}
	cluster.Name = name

	config := &types.ClusterConfigInfoEx{}
	cluster.ConfigurationEx = config

	config.DrsConfig.Enabled = types.NewBool(true)

	pool := NewResourcePool()
	Map.PutEntity(cluster, Map.NewEntity(pool))
	cluster.ResourcePool = &pool.Self

	f.putChild(cluster)
	pool.Owner = cluster.Self

	return cluster, nil
}
