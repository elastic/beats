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
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

const ec2InstanceIdentityURI = "/2014-02-25/dynamic/instance-identity/document"

// AWS EC2 Metadata Service
var ec2MetadataFetcher = provider{
	Name: "aws-ec2",

	Local: true,

	Create: func(_ string, config *common.Config) (metadataFetcher, error) {
		ec2Schema := func(m map[string]interface{}) common.MapStr {
			out, _ := s.Schema{
				"instance":          s.Object{"id": c.Str("instanceId")},
				"machine":           s.Object{"type": c.Str("instanceType")},
				"region":            c.Str("region"),
				"availability_zone": c.Str("availabilityZone"),
				"account":           s.Object{"id": c.Str("accountId")},
				"image":             s.Object{"id": c.Str("imageId")},
			}.Apply(m)
			return out
		}

		fetcher, err := newMetadataFetcher(config, "aws", nil, metadataHost, ec2Schema, ec2InstanceIdentityURI)
		return fetcher, err
	},
}
