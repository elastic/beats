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

package util

import (
	"fmt"
	"regexp"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// GeoConfig contains geo configuration data.
type GeoConfig struct {
	Name           string `config:"name"`
	Location       string `config:"location"`
	ContinentName  string `config:"continent_name"`
	CountryName    string `config:"country_name"`
	CountryISOCode string `config:"country_iso_code"`
	RegionName     string `config:"region_name"`
	RegionISOCode  string `config:"region_iso_code"`
	CityName       string `config:"city_name"`
}

// GeoConfigToMap converts `geo` sections to a `mapstr.M`.
func GeoConfigToMap(config GeoConfig) (mapstr.M, error) {
	if len(config.Location) > 0 {
		// Regexp matching a number with an optional decimal component
		// Valid numbers: '123', '123.23', etc.
		latOrLon := `\-?\d+(\.\d+)?`

		// Regexp matching a pair of lat lon coordinates.
		// e.g. 40.123, -92.929
		locRegexp := `^\s*` + // anchor to start of string with optional whitespace
			latOrLon + // match the latitude
			`\s*\,\s*` + // match the separator. optional surrounding whitespace
			latOrLon + // match the longitude
			`\s*$` //optional whitespace then end anchor

		if m, _ := regexp.MatchString(locRegexp, config.Location); !m {
			return nil, fmt.Errorf("Invalid lat,lon  string for add_observer_metadata: %s", config.Location)
		}
	}

	geoFields := mapstr.M{
		"name":             config.Name,
		"location":         config.Location,
		"continent_name":   config.ContinentName,
		"country_name":     config.CountryName,
		"country_iso_code": config.CountryISOCode,
		"region_name":      config.RegionName,
		"region_iso_code":  config.RegionISOCode,
		"city_name":        config.CityName,
	}
	// Delete any empty values
	blankStringMatch := regexp.MustCompile(`^\s*$`)
	for k, v := range geoFields {
		vStr := v.(string)
		if blankStringMatch.MatchString(vStr) {
			delete(geoFields, k)
		}
	}

	return geoFields, nil
}
