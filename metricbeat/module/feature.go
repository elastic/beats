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

package module

import (
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/metricbeat/mb"
)

// MetricSetFeature creates a new MetricSet feature.
func MetricSetFeature(
	module, name string,
	factory mb.MetricSetFactory,
	description feature.Describer,
	options ...mb.MetricSetOption,
) *feature.Feature {
	ms := mb.NewMetricSetRegistration(name, factory, options...)
	return feature.New(namespace(module), name, ms, description)
}

// Feature creates a new Module feature.
func Feature(
	module string,
	factory mb.ModuleFactory,
	description feature.Describer,
) *feature.Feature {
	return feature.New(mb.Namespace, module, factory, description)
}

func namespace(module string) string {
	return mb.Namespace + "." + module
}
