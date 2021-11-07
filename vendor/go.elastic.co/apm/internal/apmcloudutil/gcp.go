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

package apmcloudutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

	"go.elastic.co/apm/model"
)

const (
	gcpMetadataURL = "http://metadata.google.internal/computeMetadata/v1/?recursive=true"
)

// See: https://cloud.google.com/compute/docs/storing-retrieving-metadata
func getGCPCloudMetadata(ctx context.Context, client *http.Client, out *model.Cloud) error {
	req, err := http.NewRequest("GET", gcpMetadataURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}

	var gcpMetadata struct {
		Instance struct {
			// ID may be an integer or a hex string.
			ID          interface{} `json:"id"`
			MachineType string      `json:"machineType"`
			Name        string      `json:"name"`
			Zone        string      `json:"zone"`
		} `json:"instance"`
		Project struct {
			NumericProjectID *int   `json:"numericProjectId"`
			ProjectID        string `json:"projectId"`
		} `json:"project"`
	}
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&gcpMetadata); err != nil {
		return err
	}

	out.Region, out.AvailabilityZone = splitGCPZone(gcpMetadata.Instance.Zone)
	if gcpMetadata.Instance.ID != nil || gcpMetadata.Instance.Name != "" {
		out.Instance = &model.CloudInstance{
			Name: gcpMetadata.Instance.Name,
		}
		if gcpMetadata.Instance.ID != nil {
			out.Instance.ID = fmt.Sprint(gcpMetadata.Instance.ID)
		}
	}
	if gcpMetadata.Instance.MachineType != "" {
		out.Machine = &model.CloudMachine{Type: splitGCPMachineType(gcpMetadata.Instance.MachineType)}
	}
	if gcpMetadata.Project.NumericProjectID != nil || gcpMetadata.Project.ProjectID != "" {
		out.Project = &model.CloudProject{Name: gcpMetadata.Project.ProjectID}
		if gcpMetadata.Project.NumericProjectID != nil {
			out.Project.ID = strconv.Itoa(*gcpMetadata.Project.NumericProjectID)
		}
	}
	return nil
}

func splitGCPZone(s string) (region, zone string) {
	// Format: "projects/projectnum/zones/zone"
	zone = path.Base(s)
	if sep := strings.LastIndex(zone, "-"); sep != -1 {
		region = zone[:sep]
	}
	return region, zone
}

func splitGCPMachineType(s string) string {
	// Format: projects/513326162531/machineTypes/n1-standard-1
	return path.Base(s)
}
