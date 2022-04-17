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

import "github.com/menderesk/beats/v7/libbeat/common"

// ReadHostList reads a list of hosts to connect to from an configuration
// object. If the `workers` settings is > 1, each host is duplicated in the final
// host list by the number of `workers`.
func ReadHostList(cfg *common.Config) ([]string, error) {
	config := struct {
		Hosts  []string `config:"hosts"  validate:"required"`
		Worker int      `config:"worker" validate:"min=1"`
	}{
		Worker: 1,
	}

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, err
	}

	lst := config.Hosts
	if len(lst) == 0 || config.Worker <= 1 {
		return lst, nil
	}

	// duplicate entries config.Workers times
	hosts := make([]string, 0, len(lst)*config.Worker)
	for _, entry := range lst {
		for i := 0; i < config.Worker; i++ {
			hosts = append(hosts, entry)
		}
	}

	return hosts, nil
}
