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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/kibana"
)

const (
	// OutputPermission is the permission of dashboard output files.
	OutputPermission = 0644
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
func Export(client *kibana.Client, id string) ([]byte, error) {
	return client.GetDashboard(id)
}

// ExportAllFromYml exports all dashboards found in the YML file
func ExportAllFromYml(client *kibana.Client, ymlPath string) ([][]byte, ListYML, error) {
	b, err := ioutil.ReadFile(ymlPath)
	if err != nil {
		return nil, ListYML{}, errors.Wrap(err, "error opening the list of dashboards")
	}
	var list ListYML
	err = yaml.Unmarshal(b, &list)
	if err != nil {
		return nil, ListYML{}, errors.Wrap(err, "error reading the list of dashboards")
	}
	if len(list.Dashboards) == 0 {
		return nil, ListYML{}, errors.Errorf("dashboards list is empty in file %v", ymlPath)
	}

	results, err := ExportAll(client, list)

	return results, list, err
}

// ExportAll exports all dashboards from an opened and parsed dashboards YML.
func ExportAll(client *kibana.Client, list ListYML) ([][]byte, error) {
	var results [][]byte
	for _, e := range list.Dashboards {
		result, err := Export(client, e.ID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed exporting id=%v", e.ID)
		}
		results = append(results, result)
	}
	return results, nil
}

// SaveToFile creates the required directories if needed and saves dashboard.
func SaveToFile(dashboard []byte, filename, root string, version common.Version) error {
	dashboardsPath := path.Join("_meta", "kibana", strconv.Itoa(version.Major), "dashboard")
	err := os.MkdirAll(path.Join(root, dashboardsPath), 0750)
	if err != nil {
		return err
	}

	out := filepath.Join(root, dashboardsPath, filename)

	return ioutil.WriteFile(out, dashboard, OutputPermission)
}

// SaveToFile creates the required directories if needed and saves dashboard.
func SaveToFolder(dashboard []byte, root string, version common.Version) error {
	p := path.Join(root, "_meta", "kibana", strconv.Itoa(version.Major))
	err := os.MkdirAll(p, 0750)
	if err != nil {
		return fmt.Errorf("failed to create folder ('%s') for new dashboard: %+v", p, err)
	}

	r := bufio.NewReader(bytes.NewReader(dashboard))
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return saveAsset(line, p)
			}
			return fmt.Errorf("error while reading dashboard lines: %+v", err)
		}
		err = saveAsset(line, p)
		if err != nil {
			return fmt.Errorf("error while saving dashboard asset: %+v", err)
		}
	}
}

func saveAsset(line []byte, assetRoot string) error {
	var a common.MapStr
	err := json.Unmarshal(line, &a)
	if err != nil {
		return fmt.Errorf("failed to decode dashboard asset: %+v", err)
	}

	t, err := a.GetValue("type")
	if err != nil {
		return fmt.Errorf("failed to retrieve asset type: %+v", err)
	}
	assetType, ok := t.(string)
	if !ok {
		return fmt.Errorf("asset type must be string: %+v", t)
	}
	id, err := a.GetValue("id")
	if err != nil {
		return fmt.Errorf("failed to retrieve asset id: %+v", err)
	}
	assetID, ok := id.(string)
	if !ok {
		return fmt.Errorf("asset id must be string: %+v", id)
	}
	assetFolder := filepath.Join(assetRoot, assetType)
	err = os.MkdirAll(assetFolder, 0750)
	if err != nil {
		return fmt.Errorf("failed to create folder ('%s') for asset: %+v", assetFolder, err)
	}

	out := filepath.Join(assetFolder, assetID+".json")
	assetIndented, err := json.MarshalIndent(a, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to get indented bytes: %+v", err)
	}
	return ioutil.WriteFile(out, assetIndented, OutputPermission)

}
