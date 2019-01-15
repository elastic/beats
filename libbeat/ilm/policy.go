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

package ilm

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/paths"

	"github.com/elastic/beats/libbeat/common"
)

type policy struct {
	name string
	body common.MapStr
}

func newPolicy(cfg policyCfg) (*policy, error) {
	if cfg.Path == "" {
		for _, p := range defaultPolicies {
			if p.name == cfg.Name {
				return &p, nil
			}
		}
		return nil, fmt.Errorf("no ILM policy found with this name: %s", cfg.Name)
	}

	path := paths.Resolve(paths.Config, cfg.Path)
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("error checking for ilm policy %s at path %s: %s", cfg.Name, cfg.Path, err)
	}

	logp.Info("Loading ilm policy %s from file", cfg.Name)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading ilm policy %s at path %s: %s", cfg.Name, cfg.Path, err)
	}

	var p policy
	err = json.Unmarshal(content, &p)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal json policy %s: %s", cfg.Name, err)
	}

	return &p, nil
}

var defaultPolicies = []policy{beatDefaultPolicy, deleteAfterTenDays, deleteAfterOneYear}

var beatDefaultPolicy = policy{
	name: "beatDefaultPolicy",
	body: common.MapStr{
		"policy": common.MapStr{
			"phases": common.MapStr{
				"hot": common.MapStr{
					"actions": common.MapStr{
						"rollover": common.MapStr{
							"max_size": "50gb",
							"max_age":  "30d",
						},
					},
				},
			},
		},
	},
}

var deleteAfterTenDays = policy{
	name: "deleteAfter10Days",
	body: common.MapStr{
		"policy": common.MapStr{
			"phases": common.MapStr{
				"hot": common.MapStr{
					"actions": common.MapStr{
						"rollover": common.MapStr{
							"max_size": "50gb",
							"max_age":  "1d",
						},
					},
				},
				"delete": common.MapStr{
					"min_age": "10d",
					"actions": common.MapStr{
						"delete": common.MapStr{},
					},
				},
			},
		},
	},
}

var deleteAfterOneYear = policy{
	name: "deleteAfter1Year",
	body: common.MapStr{
		"policy": common.MapStr{
			"phases": common.MapStr{
				"hot": common.MapStr{
					"actions": common.MapStr{
						"rollover": common.MapStr{
							"max_size": "50gb",
							"max_age":  "1w",
						},
					},
				},
				"delete": common.MapStr{
					"min_age": "1y",
					"actions": common.MapStr{
						"delete": common.MapStr{},
					},
				},
			},
		},
	},
}
