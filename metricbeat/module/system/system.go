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
	"sync"

	"github.com/elastic/beats/v7/libbeat/common/fleetmode"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/paths"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

var once sync.Once

func init() {
	// Register the ModuleFactory function for the "system" module.
	if err := mb.Registry.AddModule("system", NewModule); err != nil {
		panic(err)
	}
}

type HostFSConfig struct {
	HostFS string `config:"system.hostfs"`
}

// Module represents the system module
type Module struct {
	mb.BaseModule
	IsAgent bool // Looks to see if metricbeat is running under agent. Useful if we have breaking changes in one but not the other.
	HostFS  string
}

type SystemModule interface {
	GetHostFS() string
}

func NewModule(base mb.BaseModule) (mb.Module, error) {
	var hostfs string

	// If this is fleet, ignore the global path, as its not being set.
	// This is a temporary hack
	if fleetmode.Enabled() {
		partialConfig := HostFSConfig{}
		base.UnpackConfig(&partialConfig)

		if partialConfig.HostFS != "" {
			hostfs = partialConfig.HostFS
		} else {
			hostfs = "/"
		}

		logp.Info("In Fleet, using HostFS: %s", hostfs)
	} else {
		hostfs = paths.Paths.Hostfs
	}

	once.Do(func() {
		initModule(hostfs)
	})

	// set the main Path,
	if fleetmode.Enabled() && len(paths.Paths.Hostfs) < 2 {
		paths.Paths.Hostfs = hostfs
	}

	return &Module{BaseModule: base, HostFS: hostfs, IsAgent: fleetmode.Enabled()}, nil
}

func (m Module) GetHostFS() string {
	return m.HostFS
}
