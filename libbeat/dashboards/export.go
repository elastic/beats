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

package dashboards

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"

	yaml "gopkg.in/yaml.v2"

	"github.com/elastic/beats/filebeat/scripts/generator"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/kibana"
)

const (
	dashboardPerm = 0644
)

// ListYML is the yaml file which contains list of available dashboards.
type ListYML struct {
	Dashboards []YMLElement `yaml:"dashboards"`
}

// YMLElement contains the data of a dashboard:
// * its uuid in Kibana
// * filename to be saved as
type YMLElement struct {
	ID   string `yaml:"id"`
	File string `yaml:"file"`
}

// Export wraps GetDashboard call to provide a more descriptive API
func Export(client *kibana.Client, id string) (common.MapStr, error) {
	return client.GetDashboard(id)
}

// ExportAllFromYml exports all dashboards found in the YML file
func ExportAllFromYml(client *kibana.Client, ymlPath string) ([]common.MapStr, ListYML, error) {
	b, err := ioutil.ReadFile(ymlPath)
	if err != nil {
		return nil, ListYML{}, fmt.Errorf("error opening the list of dashboards: %+v", err)
	}
	var list ListYML
	err = yaml.Unmarshal(b, &list)
	if err != nil {
		return nil, ListYML{}, fmt.Errorf("error reading the list of dashboards: %+v", err)
	}

	results, err := ExportAll(client, list)

	return results, list, err
}

// ExportAll exports all dashboards from an opened and parsed dashboards YML.
func ExportAll(client *kibana.Client, list ListYML) ([]common.MapStr, error) {
	var results []common.MapStr
	for _, e := range list.Dashboards {
		result, err := Export(client, e.ID)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

// SaveToFile creates the required directories if needed and saves dashboard.
func SaveToFile(dashboard common.MapStr, filename, root string, version common.Version) error {
	dashboardsPath := "_meta/kibana/" + strconv.Itoa(version.Major) + "/dashboard"
	err := generator.CreateDirectories(root, dashboardsPath)
	if err != nil {
		return err
	}

	out := filepath.Join(root, dashboardsPath, filename)
	bytes, err := json.Marshal(dashboard)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(out, bytes, dashboardPerm)
}
