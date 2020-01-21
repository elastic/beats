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

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/simulator/vpx"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

type ServiceInstance struct {
	mo.ServiceInstance
}

var serviceInstance = types.ManagedObjectReference{
	Type:  "ServiceInstance",
	Value: "ServiceInstance",
}

func NewServiceInstance(content types.ServiceContent, folder mo.Folder) *ServiceInstance {
	Map = NewRegistry()

	s := &ServiceInstance{}

	s.Self = serviceInstance
	s.Content = content

	Map.Put(s)

	f := &Folder{Folder: folder}
	Map.Put(f)

	var setting []types.BaseOptionValue

	if content.About.ApiType == "HostAgent" {
		CreateDefaultESX(f)
	} else {
		setting = vpx.Setting
	}

	objects := []object.Reference{
		NewSessionManager(*s.Content.SessionManager),
		NewPropertyCollector(s.Content.PropertyCollector),
		NewFileManager(*s.Content.FileManager),
		NewLicenseManager(*s.Content.LicenseManager),
		NewSearchIndex(*s.Content.SearchIndex),
		NewViewManager(*s.Content.ViewManager),
		NewTaskManager(*s.Content.TaskManager),
		NewOptionManager(s.Content.Setting, setting),
	}

	for _, o := range objects {
		Map.Put(o)
	}

	return s
}

func (s *ServiceInstance) RetrieveServiceContent(*types.RetrieveServiceContent) soap.HasFault {
	return &methods.RetrieveServiceContentBody{
		Res: &types.RetrieveServiceContentResponse{
			Returnval: s.Content,
		},
	}
}

func (*ServiceInstance) CurrentTime(*types.CurrentTime) soap.HasFault {
	return &methods.CurrentTimeBody{
		Res: &types.CurrentTimeResponse{
			Returnval: time.Now(),
		},
	}
}
