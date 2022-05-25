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
	"net/url"
	"strings"

	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/elastic-agent-libs/version"
)

// ModuleName is the name of this module
const (
	ModuleName = "kibana"

	// API Paths
	StatusPath         = "api/status"
	StatsPath          = "api/stats"
	ClusterRulesPath   = "api/monitoring_collection/cluster_rules"
	NodeRulesPath      = "api/monitoring_collection/node_rules"
	ClusterActionsPath = "api/monitoring_collection/cluster_actions"
	NodeActionsPath    = "api/monitoring_collection/node_actions"
	SettingsPath       = "api/settings"
)

var (
	v6_4_0 = version.MustNew("6.4.0")
	v6_5_0 = version.MustNew("6.5.0")
	v6_7_2 = version.MustNew("6.7.2")
	v7_0_0 = version.MustNew("7.0.0")
	v7_0_1 = version.MustNew("7.0.1")
	v8_2_0 = version.MustNew("8.2.0")

	// StatsAPIAvailableVersion is the version of Kibana since when the stats API is available
	StatsAPIAvailableVersion = v6_4_0

	// SettingsAPIAvailableVersion is the version of Kibana since when the settings API is available
	SettingsAPIAvailableVersion = v6_5_0

	// Version of Kibana since when the rules and task manager APIs are available
	RulesAPIAvailableVersion   = v8_2_0
	ActionsAPIAvailableVersion = v8_2_0
)

var (
	KibanaSchema = s.Schema{
		"uuid":  c.Str("uuid"),
		"name":  c.Str("name"),
		"index": c.Str("index"),
		"host": s.Object{
			"name": c.Str("host"),
		},
		"transport_address": c.Str("transport_address"),
		"version":           c.Str("version"),
		"snapshot":          c.Bool("snapshot"),
		"status":            c.Str("status"),
	}
)

func init() {
	// Register the ModuleFactory function for this module.
	if err := mb.Registry.AddModule(ModuleName, NewModule); err != nil {
		panic(err)
	}
}

// NewModule creates a new module.
func NewModule(base mb.BaseModule) (mb.Module, error) {
	xpackEnabledMetricSets := []string{
		"stats", "cluster_rules", "node_rules", "cluster_actions", "node_actions",
	}
	return elastic.NewModule(&base, xpackEnabledMetricSets, logp.NewLogger(ModuleName))
}

// GetVersion returns the version of the Kibana instance
func GetVersion(http *helper.HTTP, currentPath string) (*version.V, error) {
	content, err := fetchPath(http, currentPath, StatusPath)
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

	return version.New(status.Version.Number)
}

// IsStatsAPIAvailable returns whether the stats API is available in the given version of Kibana
func IsStatsAPIAvailable(currentKibanaVersion *version.V) bool {
	return elastic.IsFeatureAvailable(currentKibanaVersion, StatsAPIAvailableVersion)
}

// IsSettingsAPIAvailable returns whether the settings API is available in the given version of Kibana
func IsSettingsAPIAvailable(currentKibanaVersion *version.V) bool {
	return elastic.IsFeatureAvailable(currentKibanaVersion, SettingsAPIAvailableVersion)
}

// IsRulesAPIAvailable returns whether the rules API is available in the given version of Kibana
func IsRulesAPIAvailable(currentKibanaVersion *version.V) bool {
	return elastic.IsFeatureAvailable(currentKibanaVersion, RulesAPIAvailableVersion)
}

// IsActionsAPIAvailable returns whether the actions API is available in the given version of Kibana
func IsActionsAPIAvailable(currentKibanaVersion *version.V) bool {
	return elastic.IsFeatureAvailable(currentKibanaVersion, ActionsAPIAvailableVersion)
}

// IsUsageExcludable returns whether the stats API supports the exclude_usage parameter in the
// given version of Kibana
func IsUsageExcludable(currentKibanaVersion *version.V) bool {
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
