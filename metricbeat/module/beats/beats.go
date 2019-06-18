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

package beats

import (
	"encoding/json"
	"net/url"

	"github.com/elastic/beats/metricbeat/helper"
)

// ModuleName is the name of this module.
const ModuleName = "beats"

// Info construct contains the relevant data from the Beats / endpoint
type Info struct {
	UUID     string `json:"uuid"`
	Beat     string `json:"beat"`
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	Version  string `json:"version"`
}

// State construct contains the relevant data from the Beats /state endpoint
type State struct {
	Outputs struct {
		Elasticsearch struct {
			ClusterUUID string `json:"cluster_uuid"`
		} `json:"elasticsearch"`
	} `json:"outputs"`
}

// GetInfo returns the data for the Beats / endpoint.
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

// GetState returns the data for the Beats /state endpoint.
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
