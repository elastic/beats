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

package node

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/rabbitmq"
)

func init() {
	mb.Registry.MustAddMetricSet("rabbitmq", "node", New,
		mb.WithHostParser(rabbitmq.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching RabbitMQ node metrics
type MetricSet struct {
	*rabbitmq.MetricSet
}

// ClusterMetricSet is the MetricSet type used when node.collect is "all"
type ClusterMetricSet struct {
	*rabbitmq.MetricSet
}

// New creates new instance of MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	switch config.Collect {
	case configCollectNode:
		ms, err := rabbitmq.NewMetricSet(base, rabbitmq.OverviewPath)
		if err != nil {
			return nil, err
		}

		return &MetricSet{ms}, nil
	case configCollectCluster:
		ms, err := rabbitmq.NewMetricSet(base, rabbitmq.NodesPath)
		if err != nil {
			return nil, err
		}

		return &ClusterMetricSet{ms}, nil
	default:
		return nil, errors.Errorf("incorrect node.collect: %s", config.Collect)
	}
}

type apiOverview struct {
	Node string `json:"node"`
}

func (m *MetricSet) fetchOverview() (*apiOverview, error) {
	d, err := m.HTTP.FetchContent()
	if err != nil {
		return nil, err
	}

	var apiOverview apiOverview
	err = json.Unmarshal(d, &apiOverview)
	if err != nil {
		return nil, errors.Wrap(err, string(d))
	}
	return &apiOverview, nil
}

// Fetch metrics from rabbitmq node
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	o, err := m.fetchOverview()
	if err != nil {
		return errors.Wrap(err, "error in fetch")
	}

	node, err := rabbitmq.NewMetricSet(m.BaseMetricSet, rabbitmq.NodesPath+"/"+o.Node)
	if err != nil {
		return errors.Wrap(err, "error creating new metricset")
	}

	content, err := node.HTTP.FetchJSON()
	if err != nil {
		return errors.Wrap(err, "error in fetch")
	}

	evt, err := eventMapping(content)
	if err != nil {
		return errors.Wrap(err, "error in mapping")
	}
	r.Event(evt)
	return nil
}

// Fetch metrics from all rabbitmq nodes in the cluster
func (m *ClusterMetricSet) Fetch(r mb.ReporterV2) error {
	content, err := m.HTTP.FetchContent()
	if err != nil {
		return errors.Wrap(err, "error in fetch")
	}

	return eventsMapping(r, content, m)
}
