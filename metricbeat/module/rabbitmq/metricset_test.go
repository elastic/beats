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

package rabbitmq

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/rabbitmq/mtest"

	"github.com/stretchr/testify/assert"
)

func init() {
	mb.Registry.MustAddMetricSet("rabbitmq", "test", newTestMetricSet,
		mb.WithHostParser(HostParser),
	)
}

type testMetricSet struct {
	*MetricSet
}

func newTestMetricSet(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := NewMetricSet(base, "/api/overview")
	if err != nil {
		return nil, err
	}
	return &testMetricSet{ms}, nil
}

// Fetch makes an HTTP request to fetch connections metrics from the connections endpoint.
func (m *testMetricSet) Fetch() ([]common.MapStr, error) {
	_, err := m.HTTP.FetchContent()
	return nil, err
}

func TestManagementPathPrefix(t *testing.T) {
	server := mtest.Server(t, mtest.ServerConfig{
		ManagementPathPrefix: "/management_prefix",
		DataDir:              "./_meta/testdata",
	})
	defer server.Close()

	config := map[string]interface{}{
		"module":      "rabbitmq",
		"metricsets":  []string{"test"},
		"hosts":       []string{server.URL},
		pathConfigKey: "/management_prefix",
	}

	f := mbtest.NewEventsFetcher(t, config)
	_, err := f.Fetch()
	assert.NoError(t, err)
}
