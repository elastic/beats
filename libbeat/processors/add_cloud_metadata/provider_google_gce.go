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

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/libbeat/common"
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type KubeConfig struct {
	Clusters []Cluster `yaml:"clusters"`
}

type Cluster struct {
	Cluster Server `yaml:"cluster"`
}

type Server struct {
	Server string `yaml:"server"`
}

// Google GCE Metadata Service
var gceMetadataFetcher = provider{
	Name: "google-gce",

	Local: true,

	Create: func(provider string, config *common.Config) (metadataFetcher, error) {
		gceMetadataURI := "/computeMetadata/v1/?recursive=true&alt=json"
		gceHeaders := map[string]string{"Metadata-Flavor": "Google"}
		gceSchema := func(m map[string]interface{}) mapstr.M {
			cloud := mapstr.M{
				"service": mapstr.M{
					"name": "GCE",
				},
			}
			meta := mapstr.M{}

			trimLeadingPath := func(key string) {
				v, err := cloud.GetValue(key)
				if err != nil {
					return
				}
				p, ok := v.(string)
				if !ok {
					return
				}
				cloud.Put(key, path.Base(p))
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
				}.ApplyTo(cloud, instance)
				trimLeadingPath("machine.type")
				trimLeadingPath("availability_zone")
				s.Schema{
					"orchestrator": s.Object{
						"cluster": c.Dict(
							"attributes",
							s.Schema{
								"name":       c.Str("cluster-name"),
								"kubeconfig": c.Str("kubeconfig"),
							}),
					},
				}.ApplyTo(meta, instance)
			}

			if kubeconfig, err := meta.GetValue("orchestrator.cluster.kubeconfig"); err == nil {
				kubeConfig, ok := kubeconfig.(string)
				if !ok {
					meta.Delete("orchestrator.cluster.kubeconfig")
				}
				cc := &KubeConfig{}
				err := yaml.Unmarshal([]byte(kubeConfig), cc)
				if err != nil {
					meta.Delete("orchestrator.cluster.kubeconfig")
				}
				if len(cc.Clusters) > 0 {
					if cc.Clusters[0].Cluster.Server != "" {
						meta.Delete("orchestrator.cluster.kubeconfig")
						meta.Put("orchestrator.cluster.url", cc.Clusters[0].Cluster.Server)
					}
				}
			} else {
				meta.Delete("orchestrator")
			}

			if project, ok := m["project"].(map[string]interface{}); ok {
				s.Schema{
					"project": s.Object{
						"id": c.Str("projectId"),
					},
					"account": s.Object{
						"id": c.Str("projectId"),
					},
				}.ApplyTo(cloud, project)
			}

			meta.DeepUpdate(mapstr.M{"cloud": cloud})
			return meta
		}

		fetcher, err := newMetadataFetcher(config, provider, gceHeaders, metadataHost, gceSchema, gceMetadataURI)
		return fetcher, err
	},
}
