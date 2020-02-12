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

package process

import (
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/metric/system/process"
)

// Config stores the system/process config options
type Config struct {
	Procs           []string                 `config:"processes"`
	Cgroups         *bool                    `config:"process.cgroups.enabled"`
	EnvWhitelist    []string                 `config:"process.env.whitelist"`
	CacheCmdLine    bool                     `config:"process.cmdline.cache.enabled"`
	IncludeTop      process.IncludeTopConfig `config:"process.include_top_n"`
	IncludeCPUTicks bool                     `config:"process.include_cpu_ticks"`
	IncludePerCPU   bool                     `config:"process.include_per_cpu"`
	CPUTicks        *bool                    `config:"cpu_ticks"` // Deprecated
}

// Validate checks for depricated config options
func (c Config) Validate() error {
	if c.CPUTicks != nil {
		cfgwarn.Deprecate("6.1.0", "cpu_ticks is deprecated. Use process.include_cpu_ticks instead")
	}
	return nil
}

var defaultConfig = Config{
	Procs:        []string{".*"}, // collect all processes by default
	CacheCmdLine: true,
	IncludeTop: process.IncludeTopConfig{
		Enabled:  true,
		ByCPU:    0,
		ByMemory: 0,
	},
	IncludePerCPU: true,
}
