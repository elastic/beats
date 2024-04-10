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
	kubernetes "github.com/elastic/beats/v7/libbeat/processors/add_kubernetes_metadata"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// Initialize initializes all the options for the `add_kubernetes_metadata` process for metricbeat.
//
// Must be called from the settings `InitFunc`.
func Initialize() {
	// Register default indexers
	cfg := conf.NewConfig()

	//Add IP Port Indexer as a default indexer
	kubernetes.Indexing.AddDefaultIndexerConfig(kubernetes.IPPortIndexerName, *cfg)

	config := map[string]interface{}{
		"lookup_fields": []string{"service.address"},
	}
	fieldCfg, err := conf.NewConfigFrom(config)
	if err == nil {
		//Add field matcher with field to lookup as metricset.host
		kubernetes.Indexing.AddDefaultMatcherConfig(kubernetes.FieldMatcherName, *fieldCfg)
	}
}
