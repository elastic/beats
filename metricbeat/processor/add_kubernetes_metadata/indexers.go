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

package add_kubernetes_metadata

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/feature"
	kubernetes "github.com/elastic/beats/libbeat/processors/add_kubernetes_metadata"
)

// Feature expose add_kubernetes_metadata feature and overwrite the default set of configs.
var Feature = feature.MustBundle(
	kubernetes.IndexerFeature(
		kubernetes.IPPortIndexerName,
		kubernetes.NewIPPortIndexer,
		true, nil,
		feature.NewDetails(
			"IP and Port matcher",
			"Match data using the IP and Port.",
			feature.Stable,
		)),
	kubernetes.MatcherFeature(
		kubernetes.FieldMatcherName,
		kubernetes.NewFieldMatcher,
		true, mustNewFomConfig(map[string]interface{}{"lookup_fields": []string{"metricset.host"}}),
		feature.NewDetails(
			"Pod container Indexer",
			"Container indexer.",
			feature.Stable,
		)),
)

func mustNewFomConfig(m map[string]interface{}) *common.Config {
	c, err := common.NewConfigFrom(m)
	if err != nil {
		panic(err)
	}
	return c
}
