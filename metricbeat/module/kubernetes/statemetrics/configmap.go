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

package statemetrics

import (
	"github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
)

func getConfigMapMapping() (mm map[string]prometheus.MetricMap, lm map[string]prometheus.LabelMap) {

	return map[string]prometheus.MetricMap{
			"kube_configmap_info":                      prometheus.InfoMetric(),
			"kube_configmap_metadata_resource_version": prometheus.Metric("configmaprometheus.metadata.resource.version"),
			"kube_configmap_created":                   prometheus.Metric("configmaprometheus.created", prometheus.OpUnixTimestampValue()),
		},
		map[string]prometheus.LabelMap{
			"namespace":        prometheus.KeyLabel(mb.ModuleDataKey + ".namespace"),
			"configmap":        prometheus.KeyLabel("configmap"),
			"resource_version": prometheus.KeyLabel("resource_version"),
		}
}
