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

package outputs

import (
	"github.com/elastic/elastic-agent-libs/config"
)

type hostWorkerCfg struct {
	Hosts []string `config:"hosts"  validate:"required"`

	// Worker is the number of output workers desired.
	Worker int `config:"worker"`

	// Workers is an alias for Worker. If both Worker and Workers are set,
	// the value of Worker should take precedence. To always retrieve the correct
	// value, use the NumWorkers() method.
	Workers int `config:"workers"`
}

// NumWorkers returns the number of output workers desired.
func (hwc hostWorkerCfg) NumWorkers() int {
	// Both Worker and Workers are set; give precedence to Worker.
	if hwc.Worker != 0 && hwc.Workers != 0 {
		return hwc.Worker
	}

	// Only one is set; figure out which one and return its value.
	if hwc.Worker != 0 {
		return hwc.Worker
	}

	return hwc.Workers
}

// ReadHostList reads a list of hosts to connect to from an configuration
// object. If the `worker` settings is > 1, each host is duplicated in the final
// host list by the number of `worker`.
func ReadHostList(cfg *config.C) ([]string, error) {
	var config hostWorkerCfg
	err := cfg.Unpack(&config)
	if err != nil {
		return nil, err
	}

	// Default to one worker
	if config.NumWorkers() < 1 {
		config.Worker = 1
	}

	lst := config.Hosts
	if len(lst) == 0 || config.NumWorkers() <= 1 {
		return lst, nil
	}

	// duplicate entries config.NumWorkers() times
	hosts := make([]string, 0, len(lst)*config.NumWorkers())
	for _, entry := range lst {
		for i := 0; i < config.NumWorkers(); i++ {
			hosts = append(hosts, entry)
		}
	}

	return hosts, nil
}
