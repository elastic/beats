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

import "github.com/elastic/beats/libbeat/common"

// Alibaba Cloud Metadata Service
// Document https://help.aliyun.com/knowledge_detail/49122.html
var alibabaCloudMetadataFetcher = provider{
	Name: "alibaba-ecs",

	Local: false,

	Create: func(_ string, c *common.Config) (metadataFetcher, error) {
		ecsMetadataHost := "100.100.100.200"
		ecsMetadataInstanceIDURI := "/latest/meta-data/instance-id"
		ecsMetadataRegionURI := "/latest/meta-data/region-id"
		ecsMetadataZoneURI := "/latest/meta-data/zone-id"

		ecsSchema := func(m map[string]interface{}) common.MapStr {
			return common.MapStr(m)
		}

		urls, err := getMetadataURLs(c, ecsMetadataHost, []string{
			ecsMetadataInstanceIDURI,
			ecsMetadataRegionURI,
			ecsMetadataZoneURI,
		})
		if err != nil {
			return nil, err
		}
		responseHandlers := map[string]responseHandler{
			urls[0]: func(all []byte, result *result) error {
				result.metadata.Put("instance.id", string(all))
				return nil
			},
			urls[1]: func(all []byte, result *result) error {
				result.metadata["region"] = string(all)
				return nil
			},
			urls[2]: func(all []byte, result *result) error {
				result.metadata["availability_zone"] = string(all)
				return nil
			},
		}
		fetcher := &httpMetadataFetcher{"ecs", nil, responseHandlers, ecsSchema}
		return fetcher, nil
	},
}
