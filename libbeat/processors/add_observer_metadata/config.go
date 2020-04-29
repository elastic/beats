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

package add_observer_metadata

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/processors/util"
)

// Config for add_host_metadata processor.
type Config struct {
	Overwrite      bool            `config:"overwrite"`       // Overwrite if observer fields already exist
	NetInfoEnabled bool            `config:"netinfo.enabled"` // Add IP and MAC to event
	CacheTTL       time.Duration   `config:"cache.ttl"`
	Geo            *util.GeoConfig `config:"geo"`
}

func defaultConfig() Config {
	return Config{
		NetInfoEnabled: true,
		CacheTTL:       5 * time.Minute,
	}
}
