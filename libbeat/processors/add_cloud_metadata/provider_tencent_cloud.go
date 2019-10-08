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

// Tencent Cloud Metadata Service
// Document https://www.qcloud.com/document/product/213/4934
var qcloudMetadataFetcher = provider{
	Name: "tencent-qcloud",

	Local: false,

	Create: func(_ string, c *common.Config) (metadataFetcher, error) {
		qcloudMetadataHost := "metadata.tencentyun.com"
		qcloudMetadataInstanceIDURI := "/meta-data/instance-id"
		qcloudMetadataRegionURI := "/meta-data/placement/region"
		qcloudMetadataZoneURI := "/meta-data/placement/zone"

		qcloudSchema := func(m map[string]interface{}) common.MapStr {
			return common.MapStr(m)
		}

		urls, err := getMetadataURLs(c, qcloudMetadataHost, []string{
			qcloudMetadataInstanceIDURI,
			qcloudMetadataRegionURI,
			qcloudMetadataZoneURI,
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
		fetcher := &httpMetadataFetcher{"qcloud", nil, responseHandlers, qcloudSchema}
		return fetcher, nil
	},
}
