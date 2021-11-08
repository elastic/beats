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

// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import (
	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/common"
)

// Config defines the structure of heartbeat.yml.
type Config struct {
	RunOnce                 bool                 `config:"run_once"`
	RunViaSyntheticsService bool                 `config:"run_via_synthetics_service"`
	Monitors                []*common.Config     `config:"monitors"`
	ConfigMonitors          *common.Config       `config:"config.monitors"`
	Scheduler               Scheduler            `config:"scheduler"`
	Autodiscover            *autodiscover.Config `config:"autodiscover"`
	SyntheticSuites         []*common.Config     `config:"synthetic_suites"`
	Jobs                    map[string]JobLimit  `config:"jobs"`
	Service                 ServiceConfig              `config:"service"`
}

type JobLimit struct {
	Limit int64 `config:"limit" validate:"min=0"`
}

// Scheduler defines the syntax of a heartbeat.yml scheduler block.
type Scheduler struct {
	Limit    int64  `config:"limit"  validate:"min=0"`
	Location string `config:"location"`
}

type ServiceConfig struct {
	UpdateInterval string             `config:"update_interval"`
	Username       string             `config:"username"`
	Password       string             `config:"password"`
	ManifestURL    string             `config:"manifest_url"`
}

type ServiceLocation struct {
	Url string `json:"url"`

	Geo struct {
		Name     string `json:"name"`
		Location struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"location"`
	} `json:"geo"`
	Status string `json:"status"`
}

type ServiceManifest struct {
	Locations map[string]ServiceLocation  `json:"locations"`
}



// DefaultConfig is the canonical instantiation of Config.
var DefaultConfig = Config{}
