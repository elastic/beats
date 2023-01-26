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

package state_service

import (
	sm "github.com/elastic/beats/v7/metricbeat/helper/kubernetes"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"

	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

var mapping = &p.MetricsMapping{
	Metrics: map[string]p.MetricMap{
		"kube_service_info": p.InfoMetric(),
		"kube_service_labels": p.ExtendedInfoMetric(
			p.Configuration{
				StoreNonMappedLabels:     true,
				NonMappedLabelsPlacement: mb.ModuleDataKey + ".labels",
				MetricProcessingOptions:  []p.MetricOption{p.OpLabelKeyPrefixRemover("label_")},
			}),
		"kube_service_created":                      p.Metric("created", p.OpUnixTimestampValue()),
		"kube_service_spec_type":                    p.InfoMetric(),
		"kube_service_spec_external_ip":             p.InfoMetric(),
		"kube_service_status_load_balancer_ingress": p.InfoMetric(),
	},
	Labels: map[string]p.LabelMap{
		"namespace":        p.KeyLabel(mb.ModuleDataKey + ".namespace"),
		"service":          p.KeyLabel("name"),
		"type":             p.KeyLabel("type"),
		"external_ip":      p.KeyLabel("external_ip"),
		"hostname":         p.KeyLabel("ingress_hostname"),
		"ip":               p.KeyLabel("ingress_ip"),
		"cluster_ip":       p.KeyLabel("cluster_ip"),
		"external_name":    p.KeyLabel("external_name"),
		"load_balancer_ip": p.KeyLabel("load_balancer_ip"),
	},
}

// Register metricset
func init() {
	sm.Init(util.ServiceResource, mapping)
}
