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
	"path"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

// Google GCE Metadata Service
func newGceMetadataFetcher(config *common.Config) (*metadataFetcher, error) {
	gceMetadataURI := "/computeMetadata/v1/?recursive=true&alt=json"
	gceHeaders := map[string]string{"Metadata-Flavor": "Google"}
	gceSchema := func(m map[string]interface{}) common.MapStr {
		out := common.MapStr{}

		trimLeadingPath := func(key string) {
			v, err := out.GetValue(key)
			if err != nil {
				return
			}
			p, ok := v.(string)
			if !ok {
				return
			}
			out.Put(key, path.Base(p))
		}

		if instance, ok := m["instance"].(map[string]interface{}); ok {
			s.Schema{
				"instance": s.Object{
					"id":   c.StrFromNum("id"),
					"name": c.Str("name"),
				},
				"machine": s.Object{
					"type": c.Str("machineType"),
				},
				"availability_zone": c.Str("zone"),
			}.ApplyTo(out, instance)
			trimLeadingPath("machine.type")
			trimLeadingPath("availability_zone")
		}

		if project, ok := m["project"].(map[string]interface{}); ok {
			s.Schema{
				"project": s.Object{
					"id": c.Str("projectId"),
				},
			}.ApplyTo(out, project)
		}

		return out
	}

	fetcher, err := newMetadataFetcher(config, "gcp", gceHeaders, metadataHost, gceSchema, gceMetadataURI)
	return fetcher, err
}
