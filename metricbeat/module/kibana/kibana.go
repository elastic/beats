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

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"

	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

func init() {
	// Register the ModuleFactory function for this module.
	if err := mb.Registry.AddModule(ModuleName, NewModule); err != nil {
		panic(err)
	}
}

// NewModule creates a new module after performing validation.
func NewModule(base mb.BaseModule) (mb.Module, error) {
	if err := validateXPackMetricsets(base); err != nil {
		return nil, err
	}

	return &base, nil
}

// Validate that correct metricsets have been specified if xpack.enabled = true.
func validateXPackMetricsets(base mb.BaseModule) error {
	config := struct {
		Metricsets   []string `config:"metricsets"`
		XPackEnabled bool     `config:"xpack.enabled"`
	}{}
	if err := base.UnpackConfig(&config); err != nil {
		return err
	}

	// Nothing to validate if xpack.enabled != true
	if !config.XPackEnabled {
		return nil
	}

	expectedXPackMetricsets := []string{
		"stats",
	}

	if !common.MakeStringSet(config.Metricsets...).Equals(common.MakeStringSet(expectedXPackMetricsets...)) {
		return errors.Errorf("The %v module with xpack.enabled: true must have metricsets: %v", ModuleName, expectedXPackMetricsets)
	}

	return nil
}

// ModuleName is the name of this module
const ModuleName = "kibana"

var (
	v6_4_0 = common.MustNewVersion("6.4.0")
	v6_5_0 = common.MustNewVersion("6.5.0")
	v6_7_2 = common.MustNewVersion("6.7.2")
	v7_0_0 = common.MustNewVersion("7.0.0")
	v7_0_1 = common.MustNewVersion("7.0.1")

	// StatsAPIAvailableVersion is the version of Kibana since when the stats API is available
	StatsAPIAvailableVersion = v6_4_0

	// SettingsAPIAvailableVersion is the version of Kibana since when the settings API is available
	SettingsAPIAvailableVersion = v6_5_0
)

// ReportErrorForMissingField reports and returns an error message for the given
// field being missing in API response received from Kibana
func ReportErrorForMissingField(field string, r mb.ReporterV2) error {
	err := fmt.Errorf("Could not find field '%v' in Kibana stats API response", field)
	r.Error(err)
	return err
}

// GetVersion returns the version of the Kibana instance
func GetVersion(http *helper.HTTP, currentPath string) (*common.Version, error) {
	const statusPath = "api/status"
	content, err := fetchPath(http, currentPath, statusPath)
	if err != nil {
		return nil, err
	}

	var status struct {
		Version struct {
			Number string `json:"number"`
		} `json:"version"`
	}

	err = json.Unmarshal(content, &status)
	if err != nil {
		return nil, err
	}

	return common.NewVersion(status.Version.Number)
}

// IsStatsAPIAvailable returns whether the stats API is available in the given version of Kibana
func IsStatsAPIAvailable(currentKibanaVersion *common.Version) bool {
	return elastic.IsFeatureAvailable(currentKibanaVersion, StatsAPIAvailableVersion)
}

// IsSettingsAPIAvailable returns whether the settings API is available in the given version of Kibana
func IsSettingsAPIAvailable(currentKibanaVersion *common.Version) bool {
	return elastic.IsFeatureAvailable(currentKibanaVersion, SettingsAPIAvailableVersion)
}

// IsUsageExcludable returns whether the stats API supports the exclude_usage parameter in the
// given version of Kibana
func IsUsageExcludable(currentKibanaVersion *common.Version) bool {
	// (6.7.2 <= currentKibamaVersion < 7.0.0) || (7.0.1 <= currentKibanaVersion)
	return (v6_7_2.LessThanOrEqual(false, currentKibanaVersion) && currentKibanaVersion.LessThan(v7_0_0)) ||
		v7_0_1.LessThanOrEqual(false, currentKibanaVersion)
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
