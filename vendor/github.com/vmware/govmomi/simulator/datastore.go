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
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

type Datastore struct {
	mo.Datastore
}

type datastorePath struct {
	Datastore string
	Path      string
}

func parseDatastorePath(dsPath string) (*datastorePath, types.BaseMethodFault) {
	invalid := func() (*datastorePath, types.BaseMethodFault) {
		return nil, &types.InvalidDatastorePath{DatastorePath: dsPath}
	}

	if len(dsPath) == 0 {
		return invalid()
	}

	if !strings.HasPrefix(dsPath, "[") {
		return invalid()
	}
	dsPath = dsPath[1:]

	ix := strings.Index(dsPath, "]")
	if ix < 0 {
		return invalid()
	}

	return &datastorePath{
		Datastore: dsPath[:ix],
		Path:      strings.TrimSpace(dsPath[ix+1:]),
	}, nil
}

func (ds *Datastore) RefreshDatastore(*types.RefreshDatastore) soap.HasFault {
	r := &methods.RefreshDatastoreBody{}

	info := ds.Info.GetDatastoreInfo()

	// #nosec: Subprocess launching with variable
	buf, err := exec.Command("df", "-k", info.Url).Output()

	if err != nil {
		r.Fault_ = Fault(err.Error(), &types.HostConfigFault{})
		return r
	}

	lines := strings.Split(string(buf), "\n")
	columns := strings.Fields(lines[1])

	used, _ := strconv.ParseInt(columns[2], 10, 64)
	info.FreeSpace, _ = strconv.ParseInt(columns[3], 10, 64)

	info.FreeSpace *= 1024
	used *= 1024

	ds.Summary.FreeSpace = info.FreeSpace
	ds.Summary.Capacity = info.FreeSpace + used

	now := time.Now()

	info.Timestamp = &now

	return r
}
