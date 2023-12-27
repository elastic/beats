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

package core

import (
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// AddDatasetToEvent adds dataset information to the event. In particular this
// adds the module name under dataset.module.
func AddDatasetToEvent(module, metricSet string, event *mb.Event) {
	if event.RootFields == nil {
		event.RootFields = mapstr.M{}
	}

	event.RootFields.Put("event.module", module)

	// Modules without "datasets" should set their module and metricset names
	// to the same value then this will omit the event.dataset field.
	if module != metricSet {
		event.RootFields.Put("event.dataset", metricSet)
	}
}
