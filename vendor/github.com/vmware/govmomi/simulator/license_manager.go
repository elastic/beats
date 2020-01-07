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
// Copyright 2017 VMware, Inc. All Rights Reserved.
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
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

// EvalLicense is the default license
var EvalLicense = types.LicenseManagerLicenseInfo{
	LicenseKey: "00000-00000-00000-00000-00000",
	EditionKey: "eval",
	Name:       "Evaluation Mode",
	Properties: []types.KeyAnyValue{
		{
			Key: "feature",
			Value: types.KeyValue{
				Key:   "serialuri:2",
				Value: "Remote virtual Serial Port Concentrator",
			},
		},
		{
			Key: "feature",
			Value: types.KeyValue{
				Key:   "dvs",
				Value: "vSphere Distributed Switch",
			},
		},
	},
}

type LicenseManager struct {
	mo.LicenseManager
}

func NewLicenseManager(ref types.ManagedObjectReference) object.Reference {
	m := &LicenseManager{}
	m.Self = ref
	m.Licenses = []types.LicenseManagerLicenseInfo{EvalLicense}

	if Map.IsVPX() {
		am := Map.Put(&LicenseAssignmentManager{}).Reference()
		m.LicenseAssignmentManager = &am
	}

	return m
}

type LicenseAssignmentManager struct {
	mo.LicenseAssignmentManager
}

func (m *LicenseAssignmentManager) QueryAssignedLicenses(req *types.QueryAssignedLicenses) soap.HasFault {
	body := &methods.QueryAssignedLicensesBody{
		Res: &types.QueryAssignedLicensesResponse{},
	}

	// EntityId can be a HostSystem or the vCenter InstanceUuid
	if req.EntityId != "" {
		if req.EntityId != Map.content().About.InstanceUuid {
			id := types.ManagedObjectReference{
				Type:  "HostSystem",
				Value: req.EntityId,
			}

			if Map.Get(id) == nil {
				return body
			}
		}
	}

	body.Res.Returnval = []types.LicenseAssignmentManagerLicenseAssignment{
		{
			EntityId:        req.EntityId,
			AssignedLicense: EvalLicense,
		},
	}

	return body
}
