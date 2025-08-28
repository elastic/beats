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

package features

import (
	"os"
	"strings"
)

// List of input types Elasticsearch state store is enabled for
var esTypesEnabled map[string]struct{}

var isESEnabled bool

func init() {
	initFromEnv("AGENTLESS_ELASTICSEARCH_STATE_STORE_INPUT_TYPES")
}

func initFromEnv(envName string) {
	esTypesEnabled = make(map[string]struct{})

	arr := strings.Split(os.Getenv(envName), ",")
	for _, e := range arr {
		k := strings.TrimSpace(e)
		if k != "" {
			esTypesEnabled[k] = struct{}{}
		}
	}
	isESEnabled = len(esTypesEnabled) > 0
}

// IsElasticsearchStateStoreEnabled returns true if feature is enabled for agentless
func IsElasticsearchStateStoreEnabled() bool {
	return isESEnabled
}

// IsElasticsearchStateStoreEnabledForInput returns true if the provided input type uses Elasticsearch for state storage if the Elasticsearch state store feature is enabled
func IsElasticsearchStateStoreEnabledForInput(inputType string) bool {
	if IsElasticsearchStateStoreEnabled() {
		_, ok := esTypesEnabled[inputType]
		return ok
	}
	return false
}
