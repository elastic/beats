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

package beat

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
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
		"state",
		"stats",
	}

	if !common.MakeStringSet(config.Metricsets...).Equals(common.MakeStringSet(expectedXPackMetricsets...)) {
		return errors.Errorf("The %v module with xpack.enabled: true must have metricsets: %v", ModuleName, expectedXPackMetricsets)
	}

	return nil
}

// ModuleName is the name of this module.
const ModuleName = "beat"

var (
	// ErrClusterUUID is the error to be returned when the monitored beat is using the Elasticsearch output but hasn't
	// yet connected or is having trouble connecting to that Elasticsearch, so the cluster UUID cannot be
	// determined
	ErrClusterUUID = fmt.Errorf("monitored beat is using Elasticsearch output but cluster UUID cannot be determined")
)

// Info construct contains the relevant data from the Beat's / endpoint
type Info struct {
	UUID     string `json:"uuid"`
	Beat     string `json:"beat"`
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	Version  string `json:"version"`
}

// State construct contains the relevant data from the Beat's /state endpoint
type State struct {
	Monitoring struct {
		ClusterUUID string `json:"cluster_uuid"`
	} `json:"monitoring"`
	Output struct {
		Name string `json:"name"`
	} `json:"output"`
	Outputs struct {
		Elasticsearch struct {
			ClusterUUID string `json:"cluster_uuid"`
		} `json:"elasticsearch"`
	} `json:"outputs"`
}

// GetInfo returns the data for the Beat's / endpoint.
func GetInfo(m *MetricSet) (*Info, error) {
	content, err := fetchPath(m.HTTP, "/", "")
	if err != nil {
		return nil, err
	}

	info := &Info{}
	err = json.Unmarshal(content, &info)
	if err != nil {
		return nil, err
	}

	return info, nil
}

// GetState returns the data for the Beat's /state endpoint.
func GetState(m *MetricSet) (*State, error) {
	content, err := fetchPath(m.HTTP, "/state", "")
	if err != nil {
		return nil, err
	}

	info := &State{}
	err = json.Unmarshal(content, &info)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func fetchPath(httpHelper *helper.HTTP, path string, query string) ([]byte, error) {
	currentURI := httpHelper.GetURI()
	defer httpHelper.SetURI(currentURI)

	// Parses the uri to replace the path
	u, _ := url.Parse(currentURI)
	u.Path = path
	u.RawQuery = query

	// Http helper includes the HostData with username and password
	httpHelper.SetURI(u.String())
	return httpHelper.FetchContent()
}
