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
	"github.com/elastic/beats/libbeat/common"
)

const (
	osMetadataInstanceIDURI   = "/2009-04-04/meta-data/instance-id"
	osMetadataInstanceTypeURI = "/2009-04-04/meta-data/instance-type"
	osMetadataHostnameURI     = "/2009-04-04/meta-data/hostname"
	osMetadataZoneURI         = "/2009-04-04/meta-data/placement/availability-zone"
)

// newOpenstackNovaMetadataFetcher returns a metadataFetcher for the
// OpenStack Nova Metadata Service
// Document https://docs.openstack.org/nova/latest/user/metadata-service.html
func newOpenstackNovaMetadataFetcher(c *common.Config) (*metadataFetcher, error) {

	osSchema := func(m map[string]interface{}) common.MapStr {
		return common.MapStr(m)
	}

	urls, err := getMetadataURLs(c, metadataHost, []string{
		osMetadataInstanceIDURI,
		osMetadataInstanceTypeURI,
		osMetadataHostnameURI,
		osMetadataZoneURI,
	})
	if err != nil {
		return nil, err
	}

	responseHandlers := map[string]responseHandler{
		urls[0]: func(all []byte, result *result) error {
			result.metadata["instance_id"] = string(all)
			return nil
		},
		urls[1]: func(all []byte, result *result) error {
			result.metadata["machine_type"] = string(all)
			return nil
		},
		urls[2]: func(all []byte, result *result) error {
			result.metadata["instance_name"] = string(all)
			return nil
		},
		urls[3]: func(all []byte, result *result) error {
			result.metadata["availability_zone"] = string(all)
			return nil
		},
	}
	fetcher := &metadataFetcher{"openstack", nil, responseHandlers, osSchema}
	return fetcher, nil
}
