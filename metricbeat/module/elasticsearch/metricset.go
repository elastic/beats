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

package elasticsearch

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/common/productorigin"
	"github.com/menderesk/beats/v7/metricbeat/helper"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"
	pathConfigKey = "path"
)

var (
	// HostParser parses host urls for RabbitMQ management plugin
	HostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		PathConfigKey: pathConfigKey,
	}.Build()
)

type Scope int

const (
	// Indicates that each item in the hosts list points to a distinct Elasticsearch node in a
	// cluster.
	ScopeNode Scope = iota

	// Indicates that each item in the hosts lists points to a endpoint for a distinct Elasticsearch
	// cluster (e.g. a load-balancing proxy) fronting the cluster.
	ScopeCluster
)

func (h *Scope) Unpack(str string) error {
	switch str {
	case "node":
		*h = ScopeNode
	case "cluster":
		*h = ScopeCluster
	default:
		return fmt.Errorf("invalid scope: %v", str)
	}

	return nil
}

type MetricSetAPI interface {
	Module() mb.Module
	GetMasterNodeID() (string, error)
	IsMLockAllEnabled(string) (bool, error)
}

// MetricSet can be used to build other metric sets that query RabbitMQ
// management plugin
type MetricSet struct {
	mb.BaseMetricSet
	servicePath string
	*helper.HTTP
	Scope        Scope
	XPackEnabled bool
}

// NewMetricSet creates an metric set that can be used to build other metric
// sets that query RabbitMQ management plugin
func NewMetricSet(base mb.BaseMetricSet, servicePath string) (*MetricSet, error) {
	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}

	http.SetHeaderDefault(productorigin.Header, productorigin.Beats)

	config := struct {
		Scope        Scope `config:"scope"`
		XPackEnabled bool  `config:"xpack.enabled"`
	}{
		Scope:        ScopeNode,
		XPackEnabled: false,
	}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	ms := &MetricSet{
		base,
		servicePath,
		http,
		config.Scope,
		config.XPackEnabled,
	}

	ms.SetServiceURI(servicePath)

	return ms, nil
}

// GetServiceURI returns the URI of the Elasticsearch service being monitored by this metricset
func (m *MetricSet) GetServiceURI() string {
	return m.HostData().SanitizedURI + m.servicePath
}

// SetServiceURI updates the URI of the Elasticsearch service being monitored by this metricset
func (m *MetricSet) SetServiceURI(servicePath string) {
	m.servicePath = servicePath
	m.HTTP.SetURI(m.GetServiceURI())
}

func (m *MetricSet) ShouldSkipFetch() (bool, error) {
	// If we're talking to a set of ES nodes directly, only collect stats from the master node so
	// we don't collect the same stats from every node and end up duplicating them.
	if m.Scope == ScopeNode {
		isMaster, err := isMaster(m.HTTP, m.GetServiceURI())
		if err != nil {
			return false, errors.Wrap(err, "error determining if connected Elasticsearch node is master")
		}

		// Not master, no event sent
		if !isMaster {
			m.Logger().Debugf("trying to fetch %v stats from a non-master node", m.Name())
			return true, nil
		}
	}

	return false, nil
}

// GetMasterNodeID returns the ID of the Elasticsearch cluster's master node
func (m *MetricSet) GetMasterNodeID() (string, error) {
	http := m.HTTP
	resetURI := m.GetServiceURI()

	content, err := fetchPath(http, resetURI, "_nodes/_master", "filter_path=nodes.*.name")
	if err != nil {
		return "", err
	}

	var response struct {
		Nodes map[string]interface{} `json:"nodes"`
	}

	if err := json.Unmarshal(content, &response); err != nil {
		return "", err
	}

	for nodeID := range response.Nodes {
		return nodeID, nil
	}

	return "", errors.New("could not determine master node ID")
}

// IsMLockAllEnabled returns if the given Elasticsearch node has mlockall enabled
func (m *MetricSet) IsMLockAllEnabled(nodeID string) (bool, error) {
	http := m.HTTP
	resetURI := m.GetServiceURI()

	content, err := fetchPath(http, resetURI, "_nodes/"+nodeID, "filter_path=nodes.*.process.mlockall")
	if err != nil {
		return false, err
	}

	var response map[string]map[string]map[string]map[string]bool
	err = json.Unmarshal(content, &response)
	if err != nil {
		return false, err
	}

	for _, nodeInfo := range response["nodes"] {
		mlockall := nodeInfo["process"]["mlockall"]
		return mlockall, nil
	}

	return false, fmt.Errorf("could not determine if mlockall is enabled on node ID = %v", nodeID)
}
