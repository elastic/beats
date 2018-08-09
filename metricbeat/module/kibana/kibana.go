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

package kibana

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
)

const (
	// StatsAPIAvailableVersion is the version of Kibana since when the stats API is available
	StatsAPIAvailableVersion = "6.4.0"

	// SettingsAPIAvailableVersion is the version of Kibana since when the settings API is available
	SettingsAPIAvailableVersion = "6.5.0"
)

// ReportErrorForMissingField reports and returns an error message for the given
// field being missing in API response received from Kibana
func ReportErrorForMissingField(field string, r mb.ReporterV2) error {
	err := fmt.Errorf("Could not find field '%v' in Kibana stats API response", field)
	r.Error(err)
	return err
}

// GetVersion returns the version of the Kibana instance
func GetVersion(http *helper.HTTP, currentPath string) (string, error) {
	const statusPath = "api/status"
	content, err := fetchPath(http, currentPath, statusPath)
	if err != nil {
		return "", err
	}

	var data common.MapStr
	err = json.Unmarshal(content, &data)
	if err != nil {
		return "", err
	}

	version, err := data.GetValue("version.number")
	if err != nil {
		return "", err
	}

	versionStr, ok := version.(string)
	if !ok {
		return "", fmt.Errorf("Could not parse Kibana version in status API response")
	}

	return versionStr, nil
}

func isKibanaAPIAvailable(currentKibanaVersion, apiAvailableInKibanaVersion string) (bool, error) {
	currentVersion, err := common.NewVersion(currentKibanaVersion)
	if err != nil {
		return false, err
	}

	wantVersion, err := common.NewVersion(apiAvailableInKibanaVersion)
	if err != nil {
		return false, err
	}

	return !currentVersion.LessThan(wantVersion), nil
}

// IsStatsAPIAvailable returns whether the stats API is available in the given version of Kibana
func IsStatsAPIAvailable(currentKibanaVersion string) (bool, error) {
	return isKibanaAPIAvailable(currentKibanaVersion, StatsAPIAvailableVersion)
}

// IsSettingsAPIAvailable returns whether the settings API is available in the given version of Kibana
func IsSettingsAPIAvailable(currentKibanaVersion string) (bool, error) {
	return isKibanaAPIAvailable(currentKibanaVersion, SettingsAPIAvailableVersion)
}

func fetchPath(http *helper.HTTP, currentPath, newPath string) ([]byte, error) {
	currentURI := http.GetURI()
	defer http.SetURI(currentURI) // Reset after this request

	// Parse the URI to replace the path
	u, err := url.Parse(currentURI)
	if err != nil {
		return nil, err
	}

	u.Path = strings.Replace(u.Path, currentPath, newPath, 1) // HACK: to account for base paths
	u.RawQuery = ""

	// Http helper includes the HostData with username and password
	http.SetURI(u.String())
	return http.FetchContent()
}
