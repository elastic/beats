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

package apiserver

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
	k8smod "github.com/elastic/beats/v7/metricbeat/module/kubernetes"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Metricset for apiserver is a prometheus based metricset
type Metricset struct {
	mb.BaseMetricSet
	http               *helper.HTTP
	prometheusClient   prometheus.Prometheus
	prometheusMappings *prometheus.MetricsMapping
	clusterMeta        mapstr.M
	mod                k8smod.Module
}

var _ mb.ReportingMetricSetV2Error = (*Metricset)(nil)

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	pc, err := prometheus.NewPrometheusClient(base)
	if err != nil {
		return nil, err
	}

	mod, ok := base.Module().(k8smod.Module)
	if !ok {
		return nil, fmt.Errorf("must be child of kubernetes module")
	}

	http, err := pc.GetHttp()
	if err != nil {
		return nil, fmt.Errorf("the http connection is not valid")
	}
	ms := &Metricset{
		BaseMetricSet:      base,
		http:               http,
		prometheusClient:   pc,
		prometheusMappings: mapping,
		clusterMeta:        util.AddClusterECSMeta(base),
		mod:                mod,
	}

	return ms, nil
}

// Fetch gathers information from the apiserver and reports events with this information.
func (m *Metricset) Fetch(reporter mb.ReporterV2) error {
	events, err := m.prometheusClient.GetProcessedMetrics(m.prometheusMappings)
	errorString := fmt.Sprintf("%s", err)
	errorUnauthorisedMsg := fmt.Sprintf("unexpected status code %d", http.StatusUnauthorized)
	if err != nil && strings.Contains(errorString, errorUnauthorisedMsg) {
		count := 2 // We retry twice to refresh the Authorisation token in case of http.StatusUnauthorize = 401 Error
		for count > 0 {
			if _, errAuth := m.http.RefreshAuthorizationHeader(); errAuth == nil {
				events, err = m.prometheusClient.GetProcessedMetrics(m.prometheusMappings)
			}
			if err != nil {
				time.Sleep(m.mod.Config().Period)
				count--
			} else {
				break
			}
		}
	}
	// We need to check for err again in case error is not 401 or RefreshAuthorizationHeader has failed
	if err != nil {
		return fmt.Errorf("error getting metrics: %w", err)
	} else {
		for _, e := range events {
			event := mb.TransformMapStrToEvent("kubernetes", e, nil)
			if len(m.clusterMeta) != 0 {
				event.RootFields.DeepUpdate(m.clusterMeta)
			}
			isOpen := reporter.Event(event)
			if !isOpen {
				return nil
			}
		}
		return nil
	}
}
