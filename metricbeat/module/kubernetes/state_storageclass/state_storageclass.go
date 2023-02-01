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

package state_storageclass

import (
	"github.com/elastic/beats/v7/metricbeat/helper/kubernetes"
	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"
)

// mapping stores the state metrics we want to fetch and will be used by this metricset
var mapping = &p.MetricsMapping{
	Metrics: map[string]p.MetricMap{
		"kube_storageclass_info": p.InfoMetric(),
		"kube_storageclass_labels": p.ExtendedInfoMetric(
			p.Configuration{
				StoreNonMappedLabels:     true,
				NonMappedLabelsPlacement: mb.ModuleDataKey + ".labels",
				MetricProcessingOptions:  []p.MetricOption{p.OpLabelKeyPrefixRemover("label_")},
			}),
		"kube_storageclass_created": p.Metric("created", p.OpUnixTimestampValue()),
	},
	Labels: map[string]p.LabelMap{
		"storageclass":        p.KeyLabel("name"),
		"provisioner":         p.Label("provisioner"),
		"reclaimPolicy":       p.Label("reclaim_policy"),
		"reclaim_policy":      p.Label("reclaim_policy"),
		"volumeBindingMode":   p.Label("volume_binding_mode"),
		"volume_binding_mode": p.Label("volume_binding_mode"),
	},
}

// Register metricset
func init() {
	kubernetes.Init(util.StorageClassResource, mapping)
}
