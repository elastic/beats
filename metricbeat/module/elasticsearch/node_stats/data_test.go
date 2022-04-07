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

//go:build !integration
// +build !integration

package node_stats

import (
	"testing"

	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/module/elasticsearch"
)

func TestStats(t *testing.T) {
	ms := mockMetricSet{}
	elasticsearch.TestMapperWithMetricSetAndInfo(t, "./_meta/test/node_stats.*.json", ms, eventsMapping)
}

type mockMetricSet struct{}

func (m mockMetricSet) GetMasterNodeID() (string, error) {
	return "test_node_id", nil
}

func (m mockMetricSet) IsMLockAllEnabled(_ string) (bool, error) {
	return true, nil
}

func (m mockMetricSet) Module() mb.Module {
	return mockModule{}
}

type mockModule struct{}

func (m mockModule) Name() string {
	return "mock_module"
}

func (m mockModule) Config() mb.ModuleConfig {
	return mb.ModuleConfig{
		Period: 10000,
	}
}

func (m mockModule) UnpackConfig(to interface{}) error {
	return nil
}
