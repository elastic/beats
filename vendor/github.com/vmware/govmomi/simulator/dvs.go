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
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

type VmwareDistributedVirtualSwitch struct {
	mo.VmwareDistributedVirtualSwitch
}

func (s *VmwareDistributedVirtualSwitch) AddDVPortgroupTask(c *types.AddDVPortgroup_Task) soap.HasFault {
	task := CreateTask(s, "addDVPortgroup", func(t *Task) (types.AnyType, types.BaseMethodFault) {
		f := Map.getEntityParent(s, "Folder").(*Folder)

		for _, spec := range c.Spec {
			pg := &mo.DistributedVirtualPortgroup{}
			pg.Name = spec.Name
			pg.Entity().Name = pg.Name

			if obj := Map.FindByName(pg.Name, f.ChildEntity); obj != nil {
				return nil, &types.DuplicateName{
					Name:   pg.Name,
					Object: obj.Reference(),
				}
			}

			f.putChild(pg)

			pg.Key = pg.Self.Value
			pg.Config.DistributedVirtualSwitch = &s.Self

			s.Portgroup = append(s.Portgroup, pg.Self)
			s.Summary.PortgroupName = append(s.Summary.PortgroupName, pg.Name)

			for _, h := range s.Summary.HostMember {
				pg.Host = AddReference(h, pg.Host)
				host := Map.Get(h).(*HostSystem)
				host.Network = append(host.Network, pg.Reference())
			}
		}

		return nil, nil
	})

	task.Run()

	return &methods.AddDVPortgroup_TaskBody{
		Res: &types.AddDVPortgroup_TaskResponse{
			Returnval: task.Self,
		},
	}
}

func (s *VmwareDistributedVirtualSwitch) ReconfigureDvsTask(req *types.ReconfigureDvs_Task) soap.HasFault {
	task := CreateTask(s, "reconfigureDvs", func(t *Task) (types.AnyType, types.BaseMethodFault) {
		spec := req.Spec.GetDVSConfigSpec()

		for _, member := range spec.Host {
			h := Map.Get(member.Host)
			if h == nil {
				return nil, &types.ManagedObjectNotFound{Obj: member.Host}
			}

			host := h.(*HostSystem)

			switch types.ConfigSpecOperation(member.Operation) {
			case types.ConfigSpecOperationAdd:
				if FindReference(host.Network, s.Self) != nil {
					return nil, &types.AlreadyExists{Name: host.Name}
				}

				host.Network = append(host.Network, s.Self)
				host.Network = append(host.Network, s.Portgroup...)
				s.Summary.HostMember = append(s.Summary.HostMember, member.Host)

				for _, ref := range s.Portgroup {
					pg := Map.Get(ref).(*mo.DistributedVirtualPortgroup)
					pg.Host = AddReference(member.Host, pg.Host)
				}
			case types.ConfigSpecOperationRemove:
				if pg := FindReference(host.Network, s.Portgroup...); pg != nil {
					return nil, &types.ResourceInUse{
						Type: pg.Type,
						Name: pg.Value,
					}
				}

				host.Network = RemoveReference(s.Self, host.Network)
				s.Summary.HostMember = RemoveReference(s.Self, s.Summary.HostMember)
			case types.ConfigSpecOperationEdit:
				return nil, &types.NotSupported{}
			}
		}

		return nil, nil
	})

	task.Run()

	return &methods.ReconfigureDvs_TaskBody{
		Res: &types.ReconfigureDvs_TaskResponse{
			Returnval: task.Self,
		},
	}
}
