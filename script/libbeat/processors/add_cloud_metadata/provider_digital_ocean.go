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

// DigitalOcean Metadata Service
var doMetadataFetcher = provider{
	Name: "digitalocean",

	Local: true,

	Create: func(provider string, config *common.Config) (metadataFetcher, error) {
		doSchema := func(m map[string]interface{}) common.MapStr {
			out, _ := s.Schema{
				"instance": s.Object{
					"id": c.StrFromNum("droplet_id"),
				},
				"region": c.Str("region"),
			}.Apply(m)
			return out
		}
		doMetadataURI := "/metadata/v1.json"

		fetcher, err := newMetadataFetcher(config, provider, nil, metadataHost, doSchema, doMetadataURI)
		return fetcher, err
	},
}
