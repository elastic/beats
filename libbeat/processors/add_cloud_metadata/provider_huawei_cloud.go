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

package add_cloud_metadata

import (
	"encoding/json"

	"github.com/elastic/beats/v7/libbeat/common"
	conf "github.com/elastic/elastic-agent-libs/config"
)

type hwMeta struct {
	ImageName string `json:"image_name"`
	VpcID     string `json:"vpc_id"`
}

type hwMetadata struct {
	UUID             string  `json:"uuid"`
	AvailabilityZone string  `json:"availability_zone"`
	RegionID         string  `json:"region_id"`
	Meta             *hwMeta `json:"meta"`
	ProjectID        string  `json:"project_id"`
	Name             string  `json:"name"`
}

// Huawei Cloud Metadata Service
// Document https://support.huaweicloud.com/usermanual-ecs/ecs_03_0166.html
var huaweiMetadataFetcher = provider{
	Name: "huawei-cloud",

	Local: true,

	Create: func(_ string, c *conf.C) (metadataFetcher, error) {
		metadataHost := "169.254.169.254"
		huaweiCloudMetadataJSONURI := "/openstack/latest/meta_data.json"

		huaweiCloudSchema := func(m map[string]interface{}) common.MapStr {
			m["service"] = common.MapStr{
				"name": "ECS",
			}
			return common.MapStr{"cloud": m}
		}

		urls, err := getMetadataURLs(c, metadataHost, []string{
			huaweiCloudMetadataJSONURI,
		})
		if err != nil {
			return nil, err
		}
		responseHandlers := map[string]responseHandler{
			urls[0]: func(all []byte, result *result) error {
				data := new(hwMetadata)
				err := json.Unmarshal(all, data)
				if err != nil {
					return err
				}
				result.metadata.Put("instance.id", data.UUID)
				result.metadata.Put("region", data.RegionID)
				result.metadata.Put("availability_zone", data.AvailabilityZone)
				return nil
			},
		}
		fetcher := &httpMetadataFetcher{"huawei", nil, responseHandlers, huaweiCloudSchema}
		return fetcher, nil
	},
}
