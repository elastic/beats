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

package system

import (
	"flag"
	"sync"

	"github.com/elastic/beats/v7/libbeat/common/fleetmode"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

var (
	// HostFS is an alternate mountpoint for the filesytem root, for when metricbeat is running inside a container.
	HostFS = flag.String("system.hostfs", "", "mountpoint of the host's filesystem for use in monitoring a host from within a container")
)

var once sync.Once

func init() {
	// Register the ModuleFactory function for the "system" module.
	if err := mb.Registry.AddModule("system", NewModule); err != nil {
		panic(err)
	}
}

// Module represents the system module
type Module struct {
	mb.BaseModule
	HostFS  string // Mountpoint of the host's filesystem for use in monitoring inside a container.
	IsAgent bool   // Looks to see if metricbeat is running under agent. Useful if we have breaking changes in one but not the other.
}

// NewModule instatiates the system module
func NewModule(base mb.BaseModule) (mb.Module, error) {
	// This only needs to be configured once for all system modules.
	once.Do(func() {
		initModule()
	})

	return &Module{BaseModule: base, HostFS: *HostFS, IsAgent: fleetmode.Enabled()}, nil
}
