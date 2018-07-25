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

package ml_job

import (
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

func init() {
	mb.Registry.MustAddMetricSet("elasticsearch", "ml_job", New,
		mb.WithHostParser(elasticsearch.HostParser),
		mb.WithNamespace("elasticsearch.ml.job"),
	)
}

const (
	jobPath = "/_xpack/ml/anomaly_detectors/_all/_stats"
)

// MetricSet for ml job
type MetricSet struct {
	*elasticsearch.MetricSet
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The elasticsearch ml_job metricset is beta.")

	// Get the stats from the local node
	ms, err := elasticsearch.NewMetricSet(base, jobPath)
	if err != nil {
		return nil, err
	}
	return &MetricSet{MetricSet: ms}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
func (m *MetricSet) Fetch(r mb.ReporterV2) {

	isMaster, err := elasticsearch.IsMaster(m.HTTP, m.HostData().SanitizedURI+jobPath)
	if err != nil {
		r.Error(err)
		return
	}

	// Not master, no event sent
	if !isMaster {
		logp.Debug("elasticsearch", "Trying to fetch machine learning job stats from a non-master node.")
		return
	}

	content, err := m.HTTP.FetchContent()
	if err != nil {
		r.Error(err)
		return
	}

	err = eventsMapping(r, content)
	if err != nil {
		r.Error(err)
		return
	}
}
