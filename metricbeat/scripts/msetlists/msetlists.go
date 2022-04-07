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

package msetlists

import (
	"strings"

	"github.com/elastic/beats/v8/metricbeat/mb"
)

// DefaultMetricsets returns a JSON array of all registered default metricsets
// It depends upon the calling library to actually import or register the metricsets.
func DefaultMetricsets() map[string][]string {
	// List all registered modules and metricsets.
	var defaultMap = make(map[string][]string)
	for _, mod := range mb.Registry.Modules() {
		metricSets, err := mb.Registry.DefaultMetricSets(mod)
		if err != nil && !strings.Contains(err.Error(), "no default metricset for") {
			continue
		}
		defaultMap[mod] = metricSets
	}

	return defaultMap

}
