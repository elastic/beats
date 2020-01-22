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

	"github.com/elastic/beats/metricbeat/mb"
)

var (
	HostFS = flag.String("system.hostfs", "", "mountpoint of the host's filesystem for use in monitoring a host from within a container")
)

var once sync.Once

func init() {
	// Register the ModuleFactory function for the "system" module.
	if err := mb.Registry.AddModule("system", NewModule); err != nil {
		panic(err)
	}
}

type Module struct {
	mb.BaseModule
	HostFS string // Mountpoint of the host's filesystem for use in monitoring inside a container.
}

func NewModule(base mb.BaseModule) (mb.Module, error) {
	// This only needs to be configured once for all system modules.
	once.Do(func() {
		initModule()
	})

	return &Module{BaseModule: base, HostFS: *HostFS}, nil
}
