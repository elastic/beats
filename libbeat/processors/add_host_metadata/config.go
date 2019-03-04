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

package add_host_metadata

import (
	"time"
)

// Config for add_host_metadata processor.
type Config struct {
	NetInfoEnabled bool          `config:"netinfo.enabled"` // Add IP and MAC to event
	CacheTTL       time.Duration `config:"cache.ttl"`
	Geo            *GeoConfig    `config:"geo"`
	Name           string        `config:"name"`
}

// GeoConfig contains geo configuration data.
type GeoConfig struct {
	Name           string `config:"name"`
	Location       string `config:"location"`
	ContinentName  string `config:"continent_name"`
	CountryISOCode string `config:"country_iso_code"`
	RegionName     string `config:"region_name"`
	RegionISOCode  string `config:"region_iso_code"`
	CityName       string `config:"city_name"`
}

func defaultConfig() Config {
	return Config{
		NetInfoEnabled: false,
		CacheTTL:       5 * time.Minute,
	}
}
